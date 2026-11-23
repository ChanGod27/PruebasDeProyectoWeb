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
// 4. OPERACIONES DE HORAS (db.HorasTrabajada)
// ==========================================

func parseTimestampISO(valor string) (pgtype.Timestamp, error) {
	valor = strings.TrimSpace(valor)
	t, err := time.Parse(time.RFC3339Nano, valor)
	if err != nil {
		return pgtype.Timestamp{}, err
	}
	return pgtype.Timestamp{Time: t.In(time.Local), Valid: true}, nil
}

// CreateHorasTrabajadas registra una sesión del cronómetro. Soporta formulario HTML/HTMX
// (#timer-save-form en calendario.templ, validando sesión y propiedad) y JSON (API REST).
func CreateHorasTrabajadas(w http.ResponseWriter, r *http.Request, q *db.Queries, rdb *redis.Client) {
	var newHoras db.CreateHorasTrabajadasParams

	// 1. Caso envío desde Formulario HTML o HTMX (módulo cronómetro en calendario.templ)
	if strings.Contains(r.Header.Get("Content-Type"), "application/x-www-form-urlencoded") {
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

		// Seguridad (ownership): verificar que la tarea pertenezca a una materia del usuario logueado.
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
		return
	}

	// 2. Caso petición JSON (para API REST y pruebas automatizadas)
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

// DeleteHorasTrabajadas elimina un registro de horas trabajadas.
// Si proviene de HTMX, valida sesión en Redis, verifica propiedad y devuelve el calendario actualizado.
func DeleteHorasTrabajadas(w http.ResponseWriter, r *http.Request, q *db.Queries, rdb *redis.Client) {
	idStr := r.PathValue("id")

	parsed, err := strconv.ParseInt(idStr, 10, 32)
	if err != nil || parsed <= 0 {
		http.Error(w, "El ID debe ser un número entero de 32 bits válido", http.StatusBadRequest)
		return
	}
	idHoras := int32(parsed)

	if r.Header.Get("HX-Request") == "true" {
		idUsuario, err := ObtenerUsuarioIDRedis(r.Context(), r, rdb)
		if err != nil {
			RedirigirSinSesion(w, r)
			return
		}

		// Seguridad (ownership): confirmar que el registro de horas pertenezca a una tarea del usuario.
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
		return
	}

	// Caso API REST tradicional
	err = q.DeleteHorasTrabajadas(r.Context(), idHoras)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
