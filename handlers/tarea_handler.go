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
// OPERACIONES DE TAREA
// ==========================================

// convierte una cadena con formato "YYYY-MM-DDTHH:MM"
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

// operacion princiapl de creacion de tarea
func CreateTarea(w http.ResponseWriter, r *http.Request, q *db.Queries, rdb *redis.Client) {
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

	// se verifica que la materia pertenezca al usuario logueado.
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
}

// actualiacion de la tarea
func PutUpdateTarea(w http.ResponseWriter, r *http.Request, q *db.Queries, rdb *redis.Client) {
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

	pertenece, err := materiaPerteneceAUsuario(r.Context(), q, tarea.IDLista, idUsuario)
	if err != nil {
		log.Printf("ERROR BD al validar propiedad de tarea #%d: %v", idTarea, err)
		http.Error(w, "Error al validar permisos de la tarea", http.StatusInternalServerError)
		return
	}
	if !pertenece {
		http.Error(w, "No tienes permiso para editar esta tarea", http.StatusForbidden)
		return
	}

	if err := r.ParseForm(); err != nil {
		http.Error(w, "Error al procesar el formulario de edición", http.StatusBadRequest)
		return
	}

	titulo := strings.TrimSpace(r.FormValue("titulo"))
	if titulo == "" {
		http.Error(w, "El título de la tarea es obligatorio", http.StatusBadRequest)
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
	completadaVal := strings.TrimSpace(r.FormValue("completada"))
	completada := completadaVal == "on" || completadaVal == "true"

	_, err = q.UpdateTarea(r.Context(), db.UpdateTareaParams{
		IDTarea:           idTarea,
		Titulo:            titulo,
		FechaInicio:       fechaInicio,
		FechaFinalizacion: fechaFin,
		Descripcion:       pgtype.Text{String: descripcion, Valid: descripcion != ""},
		Completada:        completada,
	})
	if err != nil {
		log.Printf("ERROR BD al actualizar tarea #%d: %v", idTarea, err)
		http.Error(w, "Error al actualizar la tarea", http.StatusInternalServerError)
		return
	}

	renderMateriasActualizadas(w, r, q, rdb, idUsuario)
}

// se elimina una tarea (y sus horas trabajadas asociadas).
func DeleteTarea(w http.ResponseWriter, r *http.Request, q *db.Queries, rdb *redis.Client) {
	idStr := r.PathValue("id")

	parsed, err := strconv.ParseInt(idStr, 10, 32)
	if err != nil || parsed <= 0 {
		http.Error(w, "El ID debe ser un número entero de 32 bits válido", http.StatusBadRequest)
		return
	}

	idTarea := int32(parsed)

	idUsuario, err := ObtenerUsuarioIDRedis(r.Context(), r, rdb)
	if err != nil {
		RedirigirSinSesion(w, r)
		return
	}

	tarea, err := q.GetTareaByID(r.Context(), idTarea)
	if err != nil {
		http.Error(w, "Tarea no encontrada", http.StatusNotFound)
		return
	}

	pertenece, err := materiaPerteneceAUsuario(r.Context(), q, tarea.IDLista, idUsuario)
	if err != nil {
		log.Printf("ERROR BD al validar propiedad de tarea #%d: %v", idTarea, err)
		http.Error(w, "Error al validar permisos de la tarea", http.StatusInternalServerError)
		return
	}
	if !pertenece {
		http.Error(w, "No tienes permiso para eliminar esta tarea", http.StatusForbidden)
		return
	}

	if err := q.DeleteTarea(r.Context(), idTarea); err != nil {
		log.Printf("ERROR BD al eliminar tarea #%d: %v", idTarea, err)
		http.Error(w, "Error al eliminar la tarea", http.StatusInternalServerError)
		return
	}

	renderMateriasActualizadas(w, r, q, rdb, idUsuario)
}

// se marca como completada una tarea desde la vista de materias
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

	// se verifica que la materia de esta tarea pertenezca al usuario logueado.
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
