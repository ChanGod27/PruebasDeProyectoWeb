package handlers

import (
	"ServidorEntregable/db"
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
)

// ==========================================
// 4. OPERACIONES DE HORAS (db.HorasTrabajada)
// ==========================================

func CreateHorasTrabajadas(w http.ResponseWriter, r *http.Request, q *db.Queries) {
	var newHoras db.CreateHorasTrabajadasParams

	err := json.NewDecoder(r.Body).Decode(&newHoras)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if newHoras.IDTarea <= 0 {
		http.Error(w, "El ID de la tarea es obligatorio", http.StatusBadRequest)
		return
	}

	horasCreadas, err := q.CreateHorasTrabajadas(r.Context(), newHoras)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(horasCreadas)
}

func GetListHorasByTarea(w http.ResponseWriter, r *http.Request, q *db.Queries) {
	idStr := r.PathValue("id")

	id64, err := strconv.ParseInt(idStr, 10, 32)
	if err != nil {
		http.Error(w, "ID de tarea inválido", http.StatusBadRequest)
		return
	}

	id := int32(id64)

	listaHoras, err := q.ListHorasByTarea(r.Context(), id)
	if err != nil {
		http.Error(w, "Tarea para listar horas no encontrada", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(listaHoras)
}

func GetListHorasByRangoFechas(w http.ResponseWriter, r *http.Request, q *db.Queries) {
	TimestampInicio := r.URL.Query().Get("FechaInicio")
	TimestampInicio_2 := r.URL.Query().Get("FechaInicio_2")

	timestampInicio, err := time.Parse(time.RFC3339, TimestampInicio)
	if err != nil {
		http.Error(w, "Timestamp de inicio inválida", http.StatusBadRequest)
		return
	}

	timestampInicio_2, err := time.Parse(time.RFC3339, TimestampInicio_2)
	if err != nil {
		http.Error(w, "Timestamp de inicio_2 inválida", http.StatusBadRequest)
		return
	}

	params := db.ListHorasByRangoFechasParams{
		TimestampInicio:   pgtype.Timestamp{Time: timestampInicio, Valid: true},
		TimestampInicio_2: pgtype.Timestamp{Time: timestampInicio_2, Valid: true},
	}

	listaHoras, err := q.ListHorasByRangoFechas(r.Context(), params)
	if err != nil {
		http.Error(w, "Horas en rango de fechas no encontradas", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(listaHoras)
}

func PutUpdateHorasTrabajadas(w http.ResponseWriter, r *http.Request, q *db.Queries) {
	var HorasRecibidas db.UpdateHorasTrabajadasParams

	err := json.NewDecoder(r.Body).Decode(&HorasRecibidas)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if HorasRecibidas.IDHoras <= 0 {
		http.Error(w, "Todos los campos de ID son obligatorios", http.StatusBadRequest)
		return
	}

	HorasActualizadas, err := q.UpdateHorasTrabajadas(r.Context(), HorasRecibidas)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(HorasActualizadas)
}

func DeleteHorasTrabajadas(w http.ResponseWriter, r *http.Request, q *db.Queries) {
	idStr := r.PathValue("id")

	parsed, err := strconv.ParseInt(idStr, 10, 32)
	if err != nil {
		http.Error(w, "El ID debe ser un número entero de 32 bits válido", http.StatusBadRequest)
		return
	}

	idHoras := int32(parsed)

	err = q.DeleteHorasTrabajadas(r.Context(), idHoras)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
