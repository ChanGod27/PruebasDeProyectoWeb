package db

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

func TestQueries_CRUD(t *testing.T) {
	ctx := context.Background()

	connStr := "postgres://trabajo_web:carlos-luca-hermes@localhost:5432/go_tasks_db?sslmode=disable"

	conn, errOpen := AbridDB(ctx, connStr)
	if errOpen != nil {
		t.Fatalf("error crítico al conectar a la base de datos: %v", errOpen)
	}
	defer conn.Close(ctx)

	queries := New(conn)

	// Orquestación secuencial por tabla:
	var user CreateUserRow
	t.Run("Tabla_Usuario", func(t *testing.T) {
		user = testUsuario(ctx, t, queries)
	})

	var materia ListaTarea
	t.Run("Tabla_ListaTarea_Materia", func(t *testing.T) {
		materia = testMateria(ctx, t, queries, user.IDUsuario)
	})

	var tarea Tarea
	t.Run("Tabla_Tarea", func(t *testing.T) {
		tarea = testTarea(ctx, t, queries, materia.IDLista)
	})

	var horas HorasTrabajada
	t.Run("Tabla_HorasTrabajadas", func(t *testing.T) {
		horas = testHorasTrabajadas(ctx, t, queries, tarea.IDTarea)
	})

	t.Run("Limpieza_Delete_Secuencial", func(t *testing.T) {
		testLimpieza(ctx, t, queries, horas.IDHoras, tarea.IDTarea, materia.IDLista, user.IDUsuario)
	})
}

// =============================================================================
// 1. FUNCIONES AUXILIARES POR TABLA
// =============================================================================

func testUsuario(ctx context.Context, t *testing.T, q *Queries) CreateUserRow {
	// Create
	params := CreateUserParams{
		Nombre:     "Hermes",
		Apellido:   "Truelsegaard",
		Email:      "hermes@example.com",
		Contrasena: "hash_super_secreto_123",
	}
	user, err := q.CreateUser(ctx, params)
	if err != nil {
		t.Fatalf("CreateUser falló: %v", err)
	}
	if user.IDUsuario == 0 {
		t.Fatalf("CreateUser retornó un ID inválido (0)")
	}
	t.Logf("Usuario creado exitosamente con ID: %d", user.IDUsuario)

	// Get by ID
	userPorID, err := q.GetUserByID(ctx, user.IDUsuario)
	if err != nil {
		t.Fatalf("GetUserByID falló: %v", err)
	}
	if userPorID.Email != params.Email {
		t.Fatalf("GetUserByID inconsistente: esperado %s, obtenido %s", params.Email, userPorID.Email)
	}

	// Get by Email
	userPorEmail, err := q.GetUserByEmail(ctx, params.Email)
	if err != nil {
		t.Fatalf("GetUserByEmail falló: %v", err)
	}
	if userPorEmail.IDUsuario != user.IDUsuario {
		t.Fatalf("GetUserByEmail inconsistente: esperado ID %d, obtenido %d", user.IDUsuario, userPorEmail.IDUsuario)
	}

	return user
}

func testMateria(ctx context.Context, t *testing.T, q *Queries, idUsuario int32) ListaTarea {
	// Create
	params := CreateMateriaParams{
		IDUsuario: idUsuario,
		Titulo:    "Programación Web",
	}
	materia, err := q.CreateMateria(ctx, params)
	if err != nil {
		t.Fatalf("CreateMateria falló: %v", err)
	}
	if materia.IDLista == 0 {
		t.Fatalf("CreateMateria retornó un ID inválido (0)")
	}
	t.Logf("Materia creada con ID: %d", materia.IDLista)

	// List
	materias, err := q.ListMateriasByUsuario(ctx, idUsuario)
	if err != nil {
		t.Fatalf("ListMateriasByUsuario falló: %v", err)
	}
	if len(materias) != 1 {
		t.Fatalf("ListMateriasByUsuario esperaba 1 elemento, obtuvo %d", len(materias))
	}

	// Update
	updateParams := UpdateMateriaParams{
		IDLista: materia.IDLista,
		Titulo:  "Programación Web II",
	}
	materiaActualizada, err := q.UpdateMateria(ctx, updateParams)
	if err != nil {
		t.Fatalf("UpdateMateria falló: %v", err)
	}
	if materiaActualizada.Titulo != updateParams.Titulo {
		t.Fatalf("UpdateMateria no actualizó el título: esperado %s, obtenido %s", updateParams.Titulo, materiaActualizada.Titulo)
	}

	return materiaActualizada
}

func testTarea(ctx context.Context, t *testing.T, q *Queries, idLista int32) Tarea {
	ahora := time.Now().UTC().Truncate(time.Second)
	vencimiento := ahora.Add(48 * time.Hour)

	// Create
	params := CreateTareaParams{
		IDLista:           idLista,
		Titulo:            "Entrega de TP2",
		FechaInicio:       pgtype.Timestamp{Time: ahora, Valid: true},
		FechaFinalizacion: pgtype.Timestamp{Time: vencimiento, Valid: true},
		Descripcion:       pgtype.Text{String: "Implementar sqlc y testing", Valid: true},
	}
	tarea, err := q.CreateTarea(ctx, params)
	if err != nil {
		t.Fatalf("CreateTarea falló: %v", err)
	}
	if tarea.IDTarea == 0 {
		t.Fatalf("CreateTarea retornó un ID inválido (0)")
	}
	t.Logf("Tarea creada con ID: %d", tarea.IDTarea)

	// Get by ID
	tareaPorID, err := q.GetTareaByID(ctx, tarea.IDTarea)
	if err != nil {
		t.Fatalf("GetTareaByID falló: %v", err)
	}
	if tareaPorID.Titulo != params.Titulo {
		t.Fatalf("GetTareaByID inconsistente: esperado %s, obtenido %s", params.Titulo, tareaPorID.Titulo)
	}

	// List by Materia
	tareas, err := q.ListTareasByMateria(ctx, idLista)
	if err != nil {
		t.Fatalf("ListTareasByMateria falló: %v", err)
	}
	if len(tareas) != 1 {
		t.Fatalf("ListTareasByMateria esperaba 1 tarea, obtuvo %d", len(tareas))
	}

	// List by Rango Fechas
	tareasRango, err := q.ListTareasByRangoFechas(ctx, ListTareasByRangoFechasParams{
		FechaFinalizacion:   pgtype.Timestamp{Time: ahora.Add(-1 * time.Hour), Valid: true},
		FechaFinalizacion_2: pgtype.Timestamp{Time: vencimiento.Add(1 * time.Hour), Valid: true},
	})
	if err != nil {
		t.Fatalf("ListTareasByRangoFechas falló: %v", err)
	}
	if len(tareasRango) == 0 {
		t.Fatalf("ListTareasByRangoFechas esperaba al menos 1 resultado, obtuvo 0")
	}

	// Update
	updateParams := UpdateTareaParams{
		IDTarea:           tarea.IDTarea,
		Titulo:            "Entrega de TP2 - Finalizada",
		FechaInicio:       tarea.FechaInicio,
		FechaFinalizacion: tarea.FechaFinalizacion,
		Descripcion:       pgtype.Text{String: "Entrega completada con tests", Valid: true},
		Completada:        true,
	}
	tareaActualizada, err := q.UpdateTarea(ctx, updateParams)
	if err != nil {
		t.Fatalf("UpdateTarea falló: %v", err)
	}
	if !tareaActualizada.Completada || tareaActualizada.Titulo != updateParams.Titulo {
		t.Fatalf("UpdateTarea no persistió los cambios adecuadamente")
	}

	return tareaActualizada
}

func testHorasTrabajadas(ctx context.Context, t *testing.T, q *Queries, idTarea int32) HorasTrabajada {
	ahora := time.Now().UTC().Truncate(time.Second)
	inicioSesion := ahora.Add(1 * time.Hour)
	finSesion := ahora.Add(3 * time.Hour)

	// Create
	params := CreateHorasTrabajadasParams{
		IDTarea:         idTarea,
		TimestampInicio: pgtype.Timestamp{Time: inicioSesion, Valid: true},
		TimestampFin:    pgtype.Timestamp{Time: finSesion, Valid: true},
	}
	horas, err := q.CreateHorasTrabajadas(ctx, params)
	if err != nil {
		t.Fatalf("CreateHorasTrabajadas falló: %v", err)
	}
	if horas.IDHoras == 0 {
		t.Fatalf("CreateHorasTrabajadas retornó un ID inválido (0)")
	}
	t.Logf("Horas trabajadas registradas con ID: %d", horas.IDHoras)

	// List by Tarea
	horasPorTarea, err := q.ListHorasByTarea(ctx, idTarea)
	if err != nil {
		t.Fatalf("ListHorasByTarea falló: %v", err)
	}
	if len(horasPorTarea) != 1 {
		t.Fatalf("ListHorasByTarea esperaba 1 registro, obtuvo %d", len(horasPorTarea))
	}

	// List by Rango Fechas
	horasRango, err := q.ListHorasByRangoFechas(ctx, ListHorasByRangoFechasParams{
		TimestampInicio:   pgtype.Timestamp{Time: ahora, Valid: true},
		TimestampInicio_2: pgtype.Timestamp{Time: finSesion.Add(1 * time.Hour), Valid: true},
	})
	if err != nil {
		t.Fatalf("ListHorasByRangoFechas falló: %v", err)
	}
	if len(horasRango) == 0 {
		t.Fatalf("ListHorasByRangoFechas esperaba al menos 1 resultado, obtuvo 0")
	}

	// Update
	finSesionCorregido := finSesion.Add(30 * time.Minute).Truncate(time.Second)
	updateParams := UpdateHorasTrabajadasParams{
		IDHoras:         horas.IDHoras,
		TimestampInicio: horas.TimestampInicio,
		TimestampFin:    pgtype.Timestamp{Time: finSesionCorregido, Valid: true},
	}
	horasActualizadas, err := q.UpdateHorasTrabajadas(ctx, updateParams)
	if err != nil {
		t.Fatalf("UpdateHorasTrabajadas falló: %v", err)
	}
	if horasActualizadas.TimestampFin.Time.Unix() != finSesionCorregido.Unix() {
		t.Fatalf("UpdateHorasTrabajadas inconsistente: esperado %v, obtenido %v",
			finSesionCorregido, horasActualizadas.TimestampFin.Time)
	}

	return horasActualizadas
}

func testLimpieza(ctx context.Context, t *testing.T, q *Queries, idHoras int32, idTarea int32, idLista int32, idUsuario int32) {
	// 1. Delete Horas
	if err := q.DeleteHorasTrabajadas(ctx, idHoras); err != nil {
		t.Fatalf("DeleteHorasTrabajadas falló: %v", err)
	}
	horasCheck, err := q.ListHorasByTarea(ctx, idTarea)
	if err != nil {
		t.Fatalf("Error consultando horas tras borrado: %v", err)
	}
	if len(horasCheck) != 0 {
		t.Fatalf("DeleteHorasTrabajadas no eliminó el registro: quedan %d items", len(horasCheck))
	}

	// 2. Delete Tarea
	if err := q.DeleteTarea(ctx, idTarea); err != nil {
		t.Fatalf("DeleteTarea falló: %v", err)
	}
	_, err = q.GetTareaByID(ctx, idTarea)
	if !errors.Is(err, pgx.ErrNoRows) {
		t.Fatalf("GetTareaByID esperaba pgx.ErrNoRows tras el borrado, obtuvo: %v", err)
	}

	// 3. Delete Materia
	if err := q.DeleteMateria(ctx, idLista); err != nil {
		t.Fatalf("DeleteMateria falló: %v", err)
	}
	materiasCheck, err := q.ListMateriasByUsuario(ctx, idUsuario)
	if err != nil {
		t.Fatalf("Error consultando materias tras borrado: %v", err)
	}
	if len(materiasCheck) != 0 {
		t.Fatalf("DeleteMateria no eliminó el registro: quedan %d items", len(materiasCheck))
	}
}
