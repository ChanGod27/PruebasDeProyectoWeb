package handlers

import (
	"ServidorEntregable/db"
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/redis/go-redis/v9"
)

// ==========================================
// OPERACIONES DE TAREA (db.Tarea)
// ==========================================

func parseFechaDatetimeLocal(valor string) (pgtype.Timestamp, error) {
	valor = strings.TrimSpace(valor)
	if valor == "" {
		return pgtype.Timestamp{Valid: false}, nil
	}
	t, err := time.ParseInLocation("2006-01-02T15:04", valor, time.Local)
	if err != nil {
		return pgtype.Timestamp{}, err
	}
	return pgtype.Timestamp{Time: t, Valid: true}, nil
}

// CreateTarea procesa la creación de una tarea. Soporta formulario HTML/HTMX
// (#task-modal en materias.templ, validando sesión y propiedad en Redis) y JSON (API REST).
func CreateTarea(w http.ResponseWriter, r *http.Request, q *db.Queries, rdb *redis.Client) {
	var newTarea db.CreateTareaParams

	// 1. Caso envío desde Formulario HTML o HTMX
	if strings.Contains(r.Header.Get("Content-Type"), "application/x-www-form-urlencoded") {
		idUsuario, err := ObtenerUsuarioIDRedis(r.Context(), r, rdb)
		if err != nil {
			RedirigirSinSesion(w, r)
			return
		}

		if err := r.ParseForm(); err != nil {
			http.Error(w, "Error al procesar el formulario de tarea", http.StatusBadRequest)
			return
		}

		idLista64, err := strconv.ParseInt(strings.TrimSpace(r.FormValue("id_lista")), 10, 32)
		if err != nil || idLista64 <= 0 {
			http.Error(w, "Debes seleccionar una materia válida", http.StatusBadRequest)
			return
		}
		idLista := int32(idLista64)

		titulo := strings.TrimSpace(r.FormValue("titulo"))
		if titulo == "" {
			http.Error(w, "El título de la tarea es obligatorio", http.StatusBadRequest)
			return
		}

		// Seguridad (ownership): verificar que la materia pertenezca al usuario logueado.
		pertenece, err := materiaPerteneceAUsuario(r.Context(), q, idLista, idUsuario)
		if err != nil {
			log.Printf("ERROR BD al validar materia #%d: %v", idLista, err)
			http.Error(w, "Error al validar la materia", http.StatusInternalServerError)
			return
		}
		if !pertenece {
			http.Error(w, "No tienes permiso para agregar tareas a esta materia", http.StatusForbidden)
			return
		}

		fechaInicio, err := parseFechaDatetimeLocal(r.FormValue("fecha_inicio"))
		if err != nil {
			http.Error(w, "Formato de fecha de inicio inválido", http.StatusBadRequest)
			return
		}

		fechaFin, err := parseFechaDatetimeLocal(r.FormValue("fecha_finalizacion"))
		if err != nil {
			http.Error(w, "Formato de fecha de entrega inválido", http.StatusBadRequest)
			return
		}

		descripcion := strings.TrimSpace(r.FormValue("descripcion"))

		_, err = q.CreateTarea(r.Context(), db.CreateTareaParams{
			IDLista:           idLista,
			Titulo:            titulo,
			FechaInicio:       fechaInicio,
			FechaFinalizacion: fechaFin,
			Descripcion:       pgtype.Text{String: descripcion, Valid: descripcion != ""},
		})
		if err != nil {
			log.Printf("ERROR BD al crear tarea en materia #%d: %v", idLista, err)
			http.Error(w, "Error al crear la tarea", http.StatusInternalServerError)
			return
		}

		renderMateriasActualizadas(w, r, q, rdb, idUsuario)
		return
	}

	// 2. Caso petición JSON (para API REST y pruebas automatizadas)
	err := json.NewDecoder(r.Body).Decode(&newTarea)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if strings.TrimSpace(newTarea.Titulo) == "" || newTarea.IDLista <= 0 {
		http.Error(w, "Todos los campos son obligatorios y no pueden contener solo espacios", http.StatusBadRequest)
		return
	}

	tareaCreada, err := q.CreateTarea(r.Context(), newTarea)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(tareaCreada)
}

func GetTareaById(w http.ResponseWriter, r *http.Request, q *db.Queries) {
	idStr := r.PathValue("id")

	id64, err := strconv.ParseInt(idStr, 10, 32)
	if err != nil {
		http.Error(w, "ID de tarea inválido", http.StatusBadRequest)
		return
	}

	id := int32(id64)

	tarea, err := q.GetTareaByID(r.Context(), id)
	if err != nil {
		http.Error(w, "Tarea no encontrada", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(tarea)
}

func GetListTareasByMateria(w http.ResponseWriter, r *http.Request, q *db.Queries) {
	idStr := r.PathValue("id")

	id64, err := strconv.ParseInt(idStr, 10, 32)
	if err != nil {
		http.Error(w, "ID de materia inválido", http.StatusBadRequest)
		return
	}

	id := int32(id64)

	listaTareas, err := q.ListTareasByMateria(r.Context(), id)
	if err != nil {
		http.Error(w, "Materia para listar tareas no encontrada", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(listaTareas)
}

func GetListTareasByRangoFechas(w http.ResponseWriter, r *http.Request, q *db.Queries) {

	fechaInicioStr := r.URL.Query().Get("FechaFinalizacion")
	fechaFinalStr := r.URL.Query().Get("FechaFinalizacion_2")

	fechaInicio, err := time.Parse(time.RFC3339, fechaInicioStr)
	if err != nil {
		http.Error(w, "Fecha final inválida", http.StatusBadRequest)
		return
	}

	fechaFinal, err := time.Parse(time.RFC3339, fechaFinalStr)
	if err != nil {
		http.Error(w, "Fecha final_2 inválida", http.StatusBadRequest)
		return
	}

	params := db.ListTareasByRangoFechasParams{
		FechaFinalizacion:   pgtype.Timestamp{Time: fechaInicio, Valid: true},
		FechaFinalizacion_2: pgtype.Timestamp{Time: fechaFinal, Valid: true},
	}

	listaTareas, err := q.ListTareasByRangoFechas(r.Context(), params)
	if err != nil {
		http.Error(w, "Tareas en rango de fechas no encontradas", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(listaTareas)
}

func PutUpdateTarea(w http.ResponseWriter, r *http.Request, q *db.Queries) {
	var TareaRecibida db.UpdateTareaParams

	err := json.NewDecoder(r.Body).Decode(&TareaRecibida)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if strings.TrimSpace(TareaRecibida.Titulo) == "" ||
		TareaRecibida.IDTarea <= 0 {
		http.Error(w, "Todos los campos son obligatorios y no pueden contener solo espacios", http.StatusBadRequest)
		return
	}

	TareaActualizada, err := q.UpdateTarea(r.Context(), TareaRecibida)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(TareaActualizada)
}

func DeleteTarea(w http.ResponseWriter, r *http.Request, q *db.Queries) {
	idStr := r.PathValue("id")

	parsed, err := strconv.ParseInt(idStr, 10, 32)
	if err != nil {
		http.Error(w, "El ID debe ser un número entero de 32 bits válido", http.StatusBadRequest)
		return
	}

	idTarea := int32(parsed)

	err = q.DeleteTarea(r.Context(), idTarea)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// CompletarTarea atiende POST /tareas/{id}/completar desde el checklist de materias.templ.
// Verifica sesión en Redis y propiedad de la materia antes de marcar completada = true.
func CompletarTarea(w http.ResponseWriter, r *http.Request, q *db.Queries, rdb *redis.Client) {
	idUsuario, err := ObtenerUsuarioIDRedis(r.Context(), r, rdb)
	if err != nil {
		RedirigirSinSesion(w, r)
		return
	}

	id64, err := strconv.ParseInt(r.PathValue("id"), 10, 32)
	if err != nil || id64 <= 0 {
		http.Error(w, "ID de tarea inválido", http.StatusBadRequest)
		return
	}
	idTarea := int32(id64)

	tarea, err := q.GetTareaByID(r.Context(), idTarea)
	if err != nil {
		http.Error(w, "Tarea no encontrada", http.StatusNotFound)
		return
	}

	// Seguridad (ownership): verificar que la materia de esta tarea pertenezca al usuario logueado.
	pertenece, err := materiaPerteneceAUsuario(r.Context(), q, tarea.IDLista, idUsuario)
	if err != nil {
		log.Printf("ERROR BD al validar propiedad de tarea #%d: %v", idTarea, err)
		http.Error(w, "Error al validar permisos de la tarea", http.StatusInternalServerError)
		return
	}
	if !pertenece {
		http.Error(w, "No tienes permiso para modificar esta tarea", http.StatusForbidden)
		return
	}

	_, err = q.UpdateTarea(r.Context(), db.UpdateTareaParams{
		IDTarea:           tarea.IDTarea,
		Titulo:            tarea.Titulo,
		FechaInicio:       tarea.FechaInicio,
		FechaFinalizacion: tarea.FechaFinalizacion,
		Descripcion:       tarea.Descripcion,
		Completada:        true,
	})
	if err != nil {
		log.Printf("ERROR BD al completar tarea #%d: %v", idTarea, err)
		http.Error(w, "Error al completar la tarea", http.StatusInternalServerError)
		return
	}

	renderMateriasActualizadas(w, r, q, rdb, idUsuario)
}
