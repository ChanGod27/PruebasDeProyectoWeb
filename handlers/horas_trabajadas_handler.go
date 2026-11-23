package handlers

import (
	"ServidorEntregable/db"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/redis/go-redis/v9"
)

// ==========================================
// OPERACIONES DE HORAS TRABAJADAS
// ==========================================

// convierte una cadena con formato "YYYY-MM-DDTHH:MM"

func parseTimestampISO(valor string) (pgtype.Timestamp, error) {
	valor = strings.TrimSpace(valor)
	t, err := time.Parse(time.RFC3339Nano, valor)
	if err != nil {
		return pgtype.Timestamp{}, err
	}
	return pgtype.Timestamp{Time: t.In(time.Local), Valid: true}, nil
}

// operacion princiapl de creacion de horas trabajadas
func CreateHorasTrabajadas(w http.ResponseWriter, r *http.Request, q *db.Queries, rdb *redis.Client) {
	idUsuario, err := ObtenerUsuarioIDRedis(r.Context(), r, rdb)
	if err != nil {
		RedirigirSinSesion(w, r)
		return
	}

	if err := r.ParseForm(); err != nil {
		http.Error(w, "Error al procesar el formulario del cronómetro", http.StatusBadRequest)
		return
	}

	idTarea64, err := strconv.ParseInt(strings.TrimSpace(r.FormValue("id_tarea")), 10, 32)
	if err != nil || idTarea64 <= 0 {
		http.Error(w, "Debes seleccionar una tarea válida", http.StatusBadRequest)
		return
	}
	idTarea := int32(idTarea64)

	tarea, err := q.GetTareaByID(r.Context(), idTarea)
	if err != nil {
		http.Error(w, "Tarea no encontrada", http.StatusNotFound)
		return
	}

	// se verifica que la tarea pertenezca a una materia del usuario logueado.
	pertenece, err := materiaPerteneceAUsuario(r.Context(), q, tarea.IDLista, idUsuario)
	if err != nil {
		log.Printf("ERROR BD al validar propiedad de tarea #%d: %v", idTarea, err)
		http.Error(w, "Error al validar permisos sobre la tarea", http.StatusInternalServerError)
		return
	}
	if !pertenece {
		http.Error(w, "No tienes permiso para registrar horas en esta tarea", http.StatusForbidden)
		return
	}

	tsInicio, err := parseTimestampISO(r.FormValue("timestamp_inicio"))
	if err != nil {
		http.Error(w, "Timestamp de inicio inválido", http.StatusBadRequest)
		return
	}

	tsFin, err := parseTimestampISO(r.FormValue("timestamp_fin"))
	if err != nil {
		http.Error(w, "Timestamp de fin inválido", http.StatusBadRequest)
		return
	}

	_, err = q.CreateHorasTrabajadas(r.Context(), db.CreateHorasTrabajadasParams{
		IDTarea:         idTarea,
		TimestampInicio: tsInicio,
		TimestampFin:    tsFin,
	})
	if err != nil {
		log.Printf("ERROR BD al guardar horas trabajadas en tarea #%d: %v", idTarea, err)
		http.Error(w, "Error al guardar el tiempo trabajado", http.StatusInternalServerError)
		return
	}

	renderCalendarioActualizado(w, r, q, rdb, idUsuario)
}

// elimina un registro de horas trabajadas.
func DeleteHorasTrabajadas(w http.ResponseWriter, r *http.Request, q *db.Queries, rdb *redis.Client) {
	idStr := r.PathValue("id")

	parsed, err := strconv.ParseInt(idStr, 10, 32)
	if err != nil || parsed <= 0 {
		http.Error(w, "El ID debe ser un número entero de 32 bits válido", http.StatusBadRequest)
		return
	}
	idHoras := int32(parsed)

	idUsuario, err := ObtenerUsuarioIDRedis(r.Context(), r, rdb)
	if err != nil {
		RedirigirSinSesion(w, r)
		return
	}

	// se confirma que el registro de horas pertenezca a una tarea del usuario.
	materiasConTareas, err := cargarMateriasConTareas(r.Context(), q, idUsuario)
	if err != nil {
		log.Printf("ERROR BD al validar propiedad de horas #%d: %v", idHoras, err)
		http.Error(w, "Error al validar permisos sobre el registro", http.StatusInternalServerError)
		return
	}

	pertenece := false
	for _, m := range materiasConTareas {
		for _, t := range m.Tareas {
			horas, err := q.ListHorasByTarea(r.Context(), t.IDTarea)
			if err != nil {
				log.Printf("ERROR BD al listar horas de tarea #%d: %v", t.IDTarea, err)
				http.Error(w, "Error al validar permisos sobre el registro", http.StatusInternalServerError)
				return
			}
			for _, h := range horas {
				if h.IDHoras == idHoras {
					pertenece = true
					break
				}
			}
			if pertenece {
				break
			}
		}
		if pertenece {
			break
		}
	}

	if !pertenece {
		http.Error(w, "No tienes permiso para eliminar este registro de horas", http.StatusForbidden)
		return
	}

	if err := q.DeleteHorasTrabajadas(r.Context(), idHoras); err != nil {
		log.Printf("ERROR BD al eliminar horas #%d: %v", idHoras, err)
		http.Error(w, "Error al eliminar el registro de horas", http.StatusInternalServerError)
		return
	}

	renderCalendarioActualizado(w, r, q, rdb, idUsuario)
}
