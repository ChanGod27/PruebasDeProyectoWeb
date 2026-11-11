package handlers

import (
	"ServidorEntregable/db"
	"ServidorEntregable/view"
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"strings"
)

// ==========================================
// OPERACIONES DE USUARIO (API Y VISTAS TEMPL)
// ==========================================

// CreateUsuario procesa la creación de usuario. Soporta tanto el envío desde
// formulario HTML nativo (redireccionando a /usuarios?creado=ID) como JSON (API REST).
func CreateUsuario(w http.ResponseWriter, r *http.Request, q *db.Queries) {
	var newUsuario db.CreateUserParams

	// 1. Caso envío desde Formulario HTML nativo (sin JavaScript)
	if strings.Contains(r.Header.Get("Content-Type"), "application/x-www-form-urlencoded") {
		if err := r.ParseForm(); err != nil {
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			view.CreateUsuarioPage("Error al procesar el formulario de alta").Render(r.Context(), w)
			return
		}

		newUsuario.Nombre = strings.TrimSpace(r.FormValue("nombre"))
		newUsuario.Apellido = strings.TrimSpace(r.FormValue("apellido"))
		newUsuario.Email = strings.TrimSpace(r.FormValue("email"))
		newUsuario.Contrasena = strings.TrimSpace(r.FormValue("contrasena"))

		if newUsuario.Nombre == "" || newUsuario.Apellido == "" || newUsuario.Email == "" || newUsuario.Contrasena == "" {
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			view.CreateUsuarioPage("Todos los campos son obligatorios y no pueden contener solo espacios").Render(r.Context(), w)
			return
		}

		userCreated, err := q.CreateUser(r.Context(), newUsuario)
		if err != nil {
			log.Printf("ERROR BD al crear usuario: %v", err)
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			view.CreateUsuarioPage("No se pudo crear el usuario (posible email duplicado o error de BD)").Render(r.Context(), w)
			return
		}

		// Redirige de vuelta a la página de usuarios mostrando el nuevo usuario y marcándolo como sesión activa
		http.Redirect(w, r, "/usuarios?creado="+strconv.Itoa(int(userCreated.IDUsuario)), http.StatusSeeOther)
		return
	}

	// 2. Caso petición JSON (para API REST y pruebas automatizadas)
	err := json.NewDecoder(r.Body).Decode(&newUsuario)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if strings.TrimSpace(newUsuario.Nombre) == "" ||
		strings.TrimSpace(newUsuario.Apellido) == "" ||
		strings.TrimSpace(newUsuario.Email) == "" ||
		strings.TrimSpace(newUsuario.Contrasena) == "" {
		http.Error(w, "Todos los campos son obligatorios y no pueden contener solo espacios", http.StatusBadRequest)
		return
	}

	userCreated, err := q.CreateUser(r.Context(), newUsuario)
	if err != nil {
		log.Printf("ERROR BD al crear usuario: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(userCreated)
}

func GetUserByEmail(w http.ResponseWriter, r *http.Request, q *db.Queries) {
	email := r.PathValue("email")

	if strings.TrimSpace(email) == "" {
		http.Error(w, "El email no puede estar vacío", http.StatusBadRequest)
		return
	}

	newUsuario, err := q.GetUserByEmail(r.Context(), email)
	if err != nil {
		http.Error(w, "Usuario no encontrado", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(newUsuario)
}

func GetUserById(w http.ResponseWriter, r *http.Request, q *db.Queries) {
	idStr := r.PathValue("id")

	id64, err := strconv.ParseInt(idStr, 10, 32)
	if err != nil {
		http.Error(w, "ID de usuario inválido", http.StatusBadRequest)
		return
	}

	id := int32(id64)

	newUsuario, err := q.GetUserByID(r.Context(), id)
	if err != nil {
		http.Error(w, "Usuario no encontrado", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(newUsuario)
}

// ==========================================
// CONTROLADORES DE VISTAS (RENDER TEMPL)
// ==========================================

// UsuariosView carga la lista de usuarios desde la base de datos y renderiza layout.templ
func UsuariosView(w http.ResponseWriter, r *http.Request, q *db.Queries) {
	usuarios, err := q.ListUsers(r.Context())
	if err != nil {
		log.Printf("Aviso: No se pudo listar usuarios desde BD: %v", err)
		usuarios = []db.Usuario{}
	}

	var usuarioActivoID int32
	var usuarioActivoNombre string

	// Identificar si viene un usuario recién creado (?creado=ID) o activo (?activo=ID)
	creadoParam := r.URL.Query().Get("creado")
	if creadoParam == "" {
		creadoParam = r.URL.Query().Get("activo")
	}

	if creadoParam != "" {
		if id64, err := strconv.ParseInt(creadoParam, 10, 32); err == nil {
			usuarioActivoID = int32(id64)
			for _, u := range usuarios {
				if u.IDUsuario == usuarioActivoID {
					usuarioActivoNombre = u.Nombre + " (" + u.Nombre + " " + string([]rune(u.Apellido)[0:1]) + ".)"
					break
				}
			}
		}
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	view.LayoutPrincipal(usuarios, usuarioActivoID, usuarioActivoNombre).Render(r.Context(), w)
}

// CreateUsuarioView renderiza el formulario para añadir un nuevo usuario
func CreateUsuarioView(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	view.CreateUsuarioPage("").Render(r.Context(), w)
}

// LoginView renderiza la pantalla para comprobar la contraseña del usuario seleccionado
func LoginView(w http.ResponseWriter, r *http.Request) {
	email := r.URL.Query().Get("email")
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	view.LoginPage(email, "").Render(r.Context(), w)
}

// LoginSubmit valida la contraseña ingresada y redirige a la vista con el cartel del TP6
func LoginSubmit(w http.ResponseWriter, r *http.Request, q *db.Queries) {
	email := strings.TrimSpace(r.FormValue("email"))
	contrasena := strings.TrimSpace(r.FormValue("contrasena"))

	u, err := q.GetUserByEmail(r.Context(), email)
	if err != nil || u.Contrasena != contrasena {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		view.LoginPage(email, "Contraseña incorrecta o usuario no encontrado").Render(r.Context(), w)
		return
	}

	// Redirige con el nombre para mostrar el cartel de bienvenida y aviso TP6
	http.Redirect(w, r, "/tp6?nombre="+u.Nombre, http.StatusSeeOther)
}

// TP6View muestra el cartel informativo para el próximo TP6
func TP6View(w http.ResponseWriter, r *http.Request) {
	nombre := r.URL.Query().Get("nombre")
	if nombre == "" {
		nombre = "Usuario"
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	view.TP6Aviso(nombre).Render(r.Context(), w)
}
