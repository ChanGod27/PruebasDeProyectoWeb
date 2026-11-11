package handlers

import (
	"ServidorEntregable/db"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
)

// ==========================================
// OPERACIONES DE TAREA (db.Tarea)
// ==========================================

func CreateTarea(w http.ResponseWriter, r *http.Request, q *db.Queries) {
	var newTarea db.CreateTareaParams

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