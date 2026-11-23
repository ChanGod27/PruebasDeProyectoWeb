package main

import (
	"ServidorEntregable/db"
	"ServidorEntregable/handlers"
	"context"
	"fmt"
	"net/http"
)

func main() {
	// Conexion con instancia de postgreSQL
	ctx := context.Background()

	connStr := "postgres://trabajo_web:carlos-luca-hermes@localhost:5432/go_tasks_db?sslmode=disable"

	conn, errOpen := db.AbridDB(ctx, connStr)

	if errOpen != nil {
		fmt.Printf("error crítico al conectar a la base de datos: %v", errOpen)
		return
	}
	defer conn.Close(ctx)

	// Instanciar objeto de la base de datos (userRepository)
	queries := db.New(conn)

	// Conecta a la instancia  de Redis
	rdb, errRedis := db.AbrirRedis(ctx, "localhost:6379")
	if errRedis != nil {
		fmt.Printf("ADVERTENCIA: no se pudo conectar a Redis (sesiones deshabilitadas): %v\n", errRedis)
	} else {
		// Garantiza el cierre limpio del pool de conexiones al apagar el servidor.
		defer rdb.Close()
	}

	// Servidor de archivos estáticos
	http.Handle("/css/", http.StripPrefix("/css/", http.FileServer(http.Dir("./static/css"))))

	// ==========================================
	// HANDLERS
	// ==========================================

	// Formulario para crear nuevo usuario (create_usuario.templ)
	http.HandleFunc("GET /usuarios/nuevo", func(w http.ResponseWriter, r *http.Request) {
		handlers.CreateUsuarioView(w, r)
	})

	http.HandleFunc("POST /usuarios", func(w http.ResponseWriter, r *http.Request) {
		handlers.CreateUsuario(w, r, queries)
	})

	// Inicio de sesión (login.templ) — página principal y ruta /login
	http.HandleFunc("GET /{$}", func(w http.ResponseWriter, r *http.Request) {
		handlers.LoginView(w, r, rdb)
	})
	http.HandleFunc("GET /login", func(w http.ResponseWriter, r *http.Request) {
		handlers.LoginView(w, r, rdb)
	})
	http.HandleFunc("POST /login", func(w http.ResponseWriter, r *http.Request) {
		handlers.LoginSubmit(w, r, queries, rdb)
	})

	// Perfil del usuario activo (usuario.templ)
	http.HandleFunc("GET /usuario", func(w http.ResponseWriter, r *http.Request) {
		handlers.UsuarioView(w, r, queries, rdb)
	})

	// Dashboard de calendario y cronómetro (calendario.templ)
	http.HandleFunc("GET /calendario", func(w http.ResponseWriter, r *http.Request) {
		handlers.CalendarioView(w, r, queries, rdb)
	})

	// Tablero de materias y tareas (materias.templ)
	http.HandleFunc("GET /materias", func(w http.ResponseWriter, r *http.Request) {
		handlers.MateriasView(w, r, queries, rdb)
	})

	// Creacion de Materias
	http.HandleFunc("POST /materias", func(w http.ResponseWriter, r *http.Request) {
		handlers.CreateMateria(w, r, queries, rdb)
	})

	//Actualizacion de Materias
	http.HandleFunc("PUT /materias/{id}", func(w http.ResponseWriter, r *http.Request) {
		handlers.PutUpdateMateria(w, r, queries, rdb)
	})

	//Eliminacion de Materias
	http.HandleFunc("DELETE /materias/{id}", func(w http.ResponseWriter, r *http.Request) {
		handlers.DeleteMateria(w, r, queries, rdb)
	})

	//Creacion de Tareas
	http.HandleFunc("POST /tareas", func(w http.ResponseWriter, r *http.Request) {
		handlers.CreateTarea(w, r, queries, rdb)
	})

	//Completar Tareas
	http.HandleFunc("POST /tareas/{id}/completar", func(w http.ResponseWriter, r *http.Request) {
		handlers.CompletarTarea(w, r, queries, rdb)
	})

	//Actualizacion de Tareas
	http.HandleFunc("PUT /tareas/{id}", func(w http.ResponseWriter, r *http.Request) {
		handlers.PutUpdateTarea(w, r, queries, rdb)
	})

	//Eliminacion de Tareas
	http.HandleFunc("DELETE /tareas/{id}", func(w http.ResponseWriter, r *http.Request) {
		handlers.DeleteTarea(w, r, queries, rdb)
	})

	//Creacion de Horas
	http.HandleFunc("POST /horas_trabajadas", func(w http.ResponseWriter, r *http.Request) {
		handlers.CreateHorasTrabajadas(w, r, queries, rdb)
	})

	//Eliminacion de Horas
	http.HandleFunc("DELETE /horas/{id}", func(w http.ResponseWriter, r *http.Request) {
		handlers.DeleteHorasTrabajadas(w, r, queries, rdb)
	})

	// Salir de la Sesion
	http.HandleFunc("GET /logout", func(w http.ResponseWriter, r *http.Request) {
		handlers.LogoutRedisHandler(w, r, rdb)
	})

	port := ":8080"

	err := http.ListenAndServe(port, nil)
	if err != nil {
		fmt.Printf("Error al iniciar el servidor: %s\n", err)
	}
}
