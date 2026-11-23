package handlers

import (
	"ServidorEntregable/db"
	"ServidorEntregable/views"
	"log"
	"net/http"
	"strings"

	"github.com/redis/go-redis/v9"
)

// CreateUsuario procesa la creación de usuario.
func CreateUsuario(w http.ResponseWriter, r *http.Request, q *db.Queries) {
	var newUsuario db.CreateUserParams

	renderError := func(msg string) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		if r.Header.Get("HX-Request") == "true" {
			views.CreateUsuarioContent(msg).Render(r.Context(), w)
			return
		}
		views.CreateUsuarioPage(msg).Render(r.Context(), w)
	}

	if err := r.ParseForm(); err != nil {
		renderError("Error al procesar el formulario de alta")
		return
	}

	newUsuario.Nombre = strings.TrimSpace(r.FormValue("nombre"))
	newUsuario.Apellido = strings.TrimSpace(r.FormValue("apellido"))
	newUsuario.Email = strings.TrimSpace(r.FormValue("email"))
	newUsuario.Contrasena = strings.TrimSpace(r.FormValue("contrasena"))

	if newUsuario.Nombre == "" || newUsuario.Apellido == "" || newUsuario.Email == "" || newUsuario.Contrasena == "" {
		renderError("Todos los campos son obligatorios y no pueden contener solo espacios")
		return
	}

	userCreated, err := q.CreateUser(r.Context(), newUsuario)
	if err != nil {
		log.Printf("ERROR BD al crear usuario: %v", err)
		renderError("No se pudo crear el usuario (posible email duplicado o error de BD)")
		return
	}

	// Al registrarse con éxito, vuelve al login (sin iniciar sesión automáticamente).
	if r.Header.Get("HX-Request") == "true" {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Header().Set("HX-Push-Url", "/login")
		views.LoginContent(userCreated.Email, "", "Usuario creado correctamente. Inicia sesión.").Render(r.Context(), w)
		return
	}
	http.Redirect(w, r, "/login", http.StatusSeeOther)
}

// CreateUsuarioView renderiza el formulario para añadir un nuevo usuario (soporta carga completa y HTMX)
func CreateUsuarioView(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if r.Header.Get("HX-Request") == "true" {
		views.CreateUsuarioContent("").Render(r.Context(), w)
		return
	}
	views.CreateUsuarioPage("").Render(r.Context(), w)
}

// LoginView renderiza la pantalla de inicio de sesión (soporta carga completa y HTMX).
// Si el usuario ya tiene una sesión activa en Redis, lo redirige directamente a /calendario.
func LoginView(w http.ResponseWriter, r *http.Request, rdb *redis.Client) {
	// 1. Verificar si ya existe una sesión activa en Redis para evitar mostrar el login innecesariamente.
	if _, err := ObtenerUsuarioIDRedis(r.Context(), r, rdb); err == nil {
		if r.Header.Get("HX-Request") == "true" {
			w.Header().Set("HX-Redirect", "/calendario")
			w.WriteHeader(http.StatusOK)
			return
		}
		http.Redirect(w, r, "/calendario", http.StatusSeeOther)
		return
	}

	// 2. Renderizar el fragmento HTMX o la página completa según cómo llegó la petición.
	email := r.URL.Query().Get("email")
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if r.Header.Get("HX-Request") == "true" {
		views.LoginContent(email, "", "").Render(r.Context(), w)
		return
	}
	views.LoginPage(email, "", "").Render(r.Context(), w)
}

// LoginSubmit valida las credenciales contra PostgreSQL, crea la sesión en Redis
// (guardando la cookie HttpOnly go_tasks_session) y redirige a /calendario.
func LoginSubmit(w http.ResponseWriter, r *http.Request, q *db.Queries, rdb *redis.Client) {
	email := strings.TrimSpace(r.FormValue("email"))
	contrasena := strings.TrimSpace(r.FormValue("contrasena"))

	// Helper local para mostrar el error dentro de #auth-content (HTMX) o página completa.
	// En HTMX respondemos con 200 OK para que siempre reemplace el contenido con la alerta de error.
	renderLoginError := func(statusCode int, msg string) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		if r.Header.Get("HX-Request") == "true" {
			views.LoginContent(email, msg, "").Render(r.Context(), w)
			return
		}
		w.WriteHeader(statusCode)
		views.LoginPage(email, msg, "").Render(r.Context(), w)
	}

	u, err := q.GetUserByEmail(r.Context(), email)
	if err != nil || u.Contrasena != contrasena {
		renderLoginError(http.StatusUnauthorized, "Contraseña incorrecta o usuario no encontrado")
		return
	}

	// Crear sesión en Redis y enviar cookie HttpOnly al navegador.
	if err := CrearSesionRedis(r.Context(), w, rdb, u.IDUsuario); err != nil {
		log.Printf("ERROR Redis al crear sesión: %v", err)
		renderLoginError(http.StatusInternalServerError, "Error interno al iniciar sesión (Redis no disponible)")
		return
	}

	// Si el login fue por HTMX, usamos HX-Redirect para pasar de LayoutAuth a LayoutBase (/calendario).
	if r.Header.Get("HX-Request") == "true" {
		w.Header().Set("HX-Redirect", "/calendario")
		w.WriteHeader(http.StatusOK)
		return
	}

	http.Redirect(w, r, "/calendario", http.StatusSeeOther)
}

// UsuarioView renderiza el perfil con los datos del usuario autenticado en Redis.
// Soporta carga completa (LayoutBase) y navegación interna por HTMX (UsuarioContent).
func UsuarioView(w http.ResponseWriter, r *http.Request, q *db.Queries, rdb *redis.Client) {
	// 1. Verificar sesión activa en Redis; si no hay sesión o expiró, redirigir a /login.
	idUsuario, err := ObtenerUsuarioIDRedis(r.Context(), r, rdb)
	if err != nil {
		RedirigirSinSesion(w, r)
		return
	}

	// 2. Consultar los datos actuales del usuario en PostgreSQL (sin traer la contraseña).
	uRow, err := q.GetUserByID(r.Context(), idUsuario)
	if err != nil {
		log.Printf("ERROR BD al obtener usuario activo #%d: %v", idUsuario, err)
		CerrarSesionRedis(r.Context(), w, r, rdb)
		RedirigirSinSesion(w, r)
		return
	}

	// 3. Mapear GetUserByIDRow a db.Usuario (la vista solo muestra ID, nombre, apellido y email).
	usuario := db.Usuario{
		IDUsuario: uRow.IDUsuario,
		Nombre:    uRow.Nombre,
		Apellido:  uRow.Apellido,
		Email:     uRow.Email,
	}

	// 4. Devolver solo el fragmento #main-content si viene de HTMX, o la página completa si es carga directa / F5.
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if r.Header.Get("HX-Request") == "true" {
		views.UsuarioContent(usuario).Render(r.Context(), w)
		return
	}
	views.UsuarioPage(usuario).Render(r.Context(), w)
}
