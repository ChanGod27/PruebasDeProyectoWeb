package main

import (
	"ServidorEntregable/db"
	"ServidorEntregable/handlers"
	"context"
	"fmt"
	"net/http"
)

// ==========================================
// FUNCIÓN MAIN
// ==========================================

func main() {
	// Conexion con la base de datos
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

	// Servidor de archivos estáticos (CSS, íconos, etc.)
	staticDir := "./static"
	http.Handle("/css/", http.StripPrefix("/css/", http.FileServer(http.Dir("./static/css"))))
	http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir(staticDir))))

	// ==========================================
	// VISTAS TEMPL (HTML/CSS NATIVO)
	// ==========================================

	// Página principal: Lista de usuarios (layout.templ)
	http.HandleFunc("GET /{$}", func(w http.ResponseWriter, r *http.Request) {
		handlers.UsuariosView(w, r, queries)
	})
	http.HandleFunc("GET /usuarios", func(w http.ResponseWriter, r *http.Request) {
		handlers.UsuariosView(w, r, queries)
	})

	// Formulario para crear nuevo usuario (create_usuario.templ)
	http.HandleFunc("GET /usuarios/nuevo", func(w http.ResponseWriter, r *http.Request) {
		handlers.CreateUsuarioView(w, r)
	})

	// Inicio de sesión para comprobar contraseña (usuario.templ)
	http.HandleFunc("GET /login", func(w http.ResponseWriter, r *http.Request) {
		handlers.LoginView(w, r)
	})
	http.HandleFunc("POST /login", func(w http.ResponseWriter, r *http.Request) {
		handlers.LoginSubmit(w, r, queries)
	})

	// Aviso para funcionalidades de próxima entrega (TP6)
	http.HandleFunc("GET /tp6", func(w http.ResponseWriter, r *http.Request) {
		handlers.TP6View(w, r)
	})
	http.HandleFunc("GET /calendario", func(w http.ResponseWriter, r *http.Request) {
		handlers.TP6View(w, r)
	})
	http.HandleFunc("GET /materias", func(w http.ResponseWriter, r *http.Request) {
		handlers.TP6View(w, r)
	})

	// Handlers que conectan las operaciones crud
	// Usuarios
	http.HandleFunc("POST /usuarios", func(w http.ResponseWriter, r *http.Request) {
		handlers.CreateUsuario(w, r, queries)
	})
	http.HandleFunc("GET /usuarios/email/{email}", func(w http.ResponseWriter, r *http.Request) {
		handlers.GetUserByEmail(w, r, queries)
	})
	http.HandleFunc("GET /usuarios/{id}", func(w http.ResponseWriter, r *http.Request) {
		handlers.GetUserById(w, r, queries)
	})

	// Materias
	http.HandleFunc("POST /materias", func(w http.ResponseWriter, r *http.Request) {
		handlers.CreateMateria(w, r, queries)
	})
	http.HandleFunc("GET /materias/{id}", func(w http.ResponseWriter, r *http.Request) {
		handlers.GetListMateriasByUsuario(w, r, queries)
	})
	http.HandleFunc("PUT /materias/{id}", func(w http.ResponseWriter, r *http.Request) {
		handlers.PutUpdateMateria(w, r, queries)
	})
	http.HandleFunc("DELETE /materias/{id}", func(w http.ResponseWriter, r *http.Request) {
		handlers.DeleteMateria(w, r, queries)
	})

	// Tareas
	http.HandleFunc("POST /tareas", func(w http.ResponseWriter, r *http.Request) {
		handlers.CreateTarea(w, r, queries)
	})
	http.HandleFunc("GET /tareas/{id}", func(w http.ResponseWriter, r *http.Request) {
		handlers.GetTareaById(w, r, queries)
	})
	http.HandleFunc("GET /tareas/materia/{id}", func(w http.ResponseWriter, r *http.Request) {
		handlers.GetListTareasByMateria(w, r, queries)
	})
	http.HandleFunc("GET /tareas/fechas", func(w http.ResponseWriter, r *http.Request) {
		handlers.GetListTareasByRangoFechas(w, r, queries)
	})
	http.HandleFunc("PUT /tareas/{id}", func(w http.ResponseWriter, r *http.Request) {
		handlers.PutUpdateTarea(w, r, queries)
	})
	http.HandleFunc("DELETE /tareas/{id}", func(w http.ResponseWriter, r *http.Request) {
		handlers.DeleteTarea(w, r, queries)
	})

	// Horas
	http.HandleFunc("POST /horas", func(w http.ResponseWriter, r *http.Request) {
		handlers.CreateHorasTrabajadas(w, r, queries)
	})
	http.HandleFunc("GET /horas/{id}", func(w http.ResponseWriter, r *http.Request) {
		handlers.GetListHorasByTarea(w, r, queries)
	})
	http.HandleFunc("GET /horas/fechas", func(w http.ResponseWriter, r *http.Request) {
		handlers.GetListHorasByRangoFechas(w, r, queries)
	})
	http.HandleFunc("PUT /horas/{id}", func(w http.ResponseWriter, r *http.Request) {
		handlers.PutUpdateHorasTrabajadas(w, r, queries)
	})
	http.HandleFunc("DELETE /horas/{id}", func(w http.ResponseWriter, r *http.Request) {
		handlers.DeleteHorasTrabajadas(w, r, queries)
	})

	port := ":8080"
	fmt.Printf("Servidor ESTÁTICO escuchando en http://localhost%s\n", port)
	fmt.Printf("Sirviendo archivos desde: %s\n", staticDir)

	err := http.ListenAndServe(port, nil)
	if err != nil {
		fmt.Printf("Error al iniciar el servidor: %s\n", err)
	}
}
