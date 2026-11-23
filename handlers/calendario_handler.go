package handlers

import (
	"ServidorEntregable/db"
	"ServidorEntregable/views"
	"context"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/redis/go-redis/v9"
)

// ==========================================
// OPERACIONES DE CALENDARIO
// ==========================================

// se acota la hora de un evento al rango visible de la grilla (08:00 a 21:00)
func ajustarHoraGrilla(h int) int {
	if h < 8 {
		return 8
	}
	if h > 21 {
		return 21
	}
	return h
}

/*
se reconstruye la fecha/hora en time.Local manteniendo el año, mes, día,
hora y minuto exactos almacenados en PostgreSQL (TIMESTAMP sin zona horaria).
*/
func normalizarWallTime(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), t.Hour(), t.Minute(), t.Second(), 0, time.Local)
}

// se reúne las tareas y horas trabajadas de la semana actual (Lunes a Sábado)
func cargarDatosCalendario(ctx context.Context, q *db.Queries, idUsuario int32) ([]db.Tarea, []views.DiaCalendario, []views.EventoCalendario, error) {
	materiasConTareas, err := cargarMateriasConTareas(ctx, q, idUsuario)
	if err != nil {
		return nil, nil, nil, err
	}

	ahora := time.Now().In(time.Local)
	diasDesdeLunes := (int(ahora.Weekday()) + 6) % 7
	inicioSemana := time.Date(ahora.Year(), ahora.Month(), ahora.Day()-diasDesdeLunes, 0, 0, 0, 0, time.Local)
	finSemana := inicioSemana.AddDate(0, 0, 6)

	var todasTareas []db.Tarea
	var eventos []views.EventoCalendario
	eventosPorDia := make([]int, 6)

	for _, m := range materiasConTareas {
		for _, t := range m.Tareas {
			todasTareas = append(todasTareas, t)

			// 1a. Pintar la fecha de inicio de la tarea (en verde) si cae dentro de esta semana.
			if t.FechaInicio.Valid {
				fechaIni := normalizarWallTime(t.FechaInicio.Time)
				if !fechaIni.Before(inicioSemana) && fechaIni.Before(finSemana) {
					diaIdx := int(fechaIni.Sub(inicioSemana).Hours()) / 24
					if diaIdx >= 0 && diaIdx < 6 {
						eventos = append(eventos, views.EventoCalendario{
							IDTarea:      t.IDTarea,
							Tipo:         "inicio",
							Titulo:       t.Titulo,
							DiaIndex:     diaIdx,
							Hora:         ajustarHoraGrilla(fechaIni.Hour()),
							RangoHorario: fmt.Sprintf("Inicio %02d:%02d", fechaIni.Hour(), fechaIni.Minute()),
						})
						eventosPorDia[diaIdx]++
					}
				}
			}

			// 1b. Pintar la fecha de finalización de la tarea (en rojo) si cae dentro de esta semana.
			if t.FechaFinalizacion.Valid {
				fechaFin := normalizarWallTime(t.FechaFinalizacion.Time)
				if !fechaFin.Before(inicioSemana) && fechaFin.Before(finSemana) {
					diaIdx := int(fechaFin.Sub(inicioSemana).Hours()) / 24
					if diaIdx >= 0 && diaIdx < 6 {
						eventos = append(eventos, views.EventoCalendario{
							IDTarea:      t.IDTarea,
							Tipo:         "fin",
							Titulo:       t.Titulo,
							DiaIndex:     diaIdx,
							Hora:         ajustarHoraGrilla(fechaFin.Hour()),
							RangoHorario: fmt.Sprintf("Entrega %02d:%02d", fechaFin.Hour(), fechaFin.Minute()),
						})
						eventosPorDia[diaIdx]++
					}
				}
			}

			// 2. Pintar sesiones registradas con el cronómetro para esta tarea en la semana actual.
			horas, err := q.ListHorasByTarea(ctx, t.IDTarea)
			if err != nil {
				return nil, nil, nil, err
			}
			for _, h := range horas {
				if !h.TimestampInicio.Valid {
					continue
				}
				ini := normalizarWallTime(h.TimestampInicio.Time)
				if ini.Before(inicioSemana) || !ini.Before(finSemana) {
					continue
				}
				diaIdx := int(ini.Sub(inicioSemana).Hours()) / 24
				if diaIdx < 0 || diaIdx >= 6 {
					continue
				}

				rango := fmt.Sprintf("%02d:%02d", ini.Hour(), ini.Minute())
				if h.TimestampFin.Valid {
					fin := normalizarWallTime(h.TimestampFin.Time)
					rango = fmt.Sprintf("%02d:%02d - %02d:%02d", ini.Hour(), ini.Minute(), fin.Hour(), fin.Minute())
				}

				eventos = append(eventos, views.EventoCalendario{
					IDTarea:      t.IDTarea,
					IDHoras:      h.IDHoras,
					Tipo:         "horas",
					Titulo:       "⏱ " + t.Titulo,
					DiaIndex:     diaIdx,
					Hora:         ajustarHoraGrilla(ini.Hour()),
					RangoHorario: rango,
				})
				eventosPorDia[diaIdx]++
			}
		}
	}

	nombresDias := []string{"Lunes", "Martes", "Miércoles", "Jueves", "Viernes", "Sábado"}
	dias := make([]views.DiaCalendario, 6)
	for i := 0; i < 6; i++ {
		fechaDia := inicioSemana.AddDate(0, 0, i)
		esHoy := fechaDia.Year() == ahora.Year() && fechaDia.Month() == ahora.Month() && fechaDia.Day() == ahora.Day()

		resumen := "Sin eventos"
		color := "blue"
		if eventosPorDia[i] == 1 {
			resumen = "1 evento"
			color = "green"
		} else if eventosPorDia[i] > 1 {
			resumen = fmt.Sprintf("%d eventos", eventosPorDia[i])
			color = "green"
		}
		if esHoy {
			color = "amber"
		}

		dias[i] = views.DiaCalendario{
			Nombre:     nombresDias[i],
			Fecha:      fechaDia.Format("02/01"),
			Resumen:    resumen,
			ColorClass: color,
			EsHoy:      esHoy,
		}
	}

	return todasTareas, dias, eventos, nil
}

// se construye los datos de la semana
func renderCalendarioActualizado(w http.ResponseWriter, r *http.Request, q *db.Queries, rdb *redis.Client, idUsuario int32) {
	tareas, dias, eventos, err := cargarDatosCalendario(r.Context(), q, idUsuario)
	if err != nil {
		log.Printf("ERROR BD al cargar calendario del usuario #%d: %v", idUsuario, err)
		http.Error(w, "Error al cargar el calendario", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if r.Header.Get("HX-Request") == "true" {
		views.CalendarioContent(tareas, dias, eventos).Render(r.Context(), w)
		return
	}

	uRow, err := q.GetUserByID(r.Context(), idUsuario)
	if err != nil {
		log.Printf("ERROR BD al obtener usuario activo #%d: %v", idUsuario, err)
		CerrarSesionRedis(r.Context(), w, r, rdb)
		RedirigirSinSesion(w, r)
		return
	}

	usuario := db.Usuario{
		IDUsuario: uRow.IDUsuario,
		Nombre:    uRow.Nombre,
		Apellido:  uRow.Apellido,
		Email:     uRow.Email,
	}

	views.CalendarioPage(usuario, tareas, dias, eventos).Render(r.Context(), w)
}

// se renderiza el cronómetro y calendario semanal
func CalendarioView(w http.ResponseWriter, r *http.Request, q *db.Queries, rdb *redis.Client) {
	idUsuario, err := ObtenerUsuarioIDRedis(r.Context(), r, rdb)
	if err != nil {
		RedirigirSinSesion(w, r)
		return
	}

	renderCalendarioActualizado(w, r, q, rdb, idUsuario)
}
