package handlers

import (
	"ServidorEntregable/db"
	"ServidorEntregable/views"
	"context"
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"strings"

	"github.com/redis/go-redis/v9"
)

// ==========================================
// OPERACIONES DE MATERIA (db.ListaTarea)
// ==========================================

// CreateMateria procesa la creación de una materia. Soporta formulario HTML/HTMX
// (tomando el IDUsuario desde la sesión en Redis) y JSON (API REST).
func CreateMateria(w http.ResponseWriter, r *http.Request, q *db.Queries, rdb *redis.Client) {
	var newMateria db.CreateMateriaParams

	// 1. Caso envío desde Formulario HTML o HTMX (#materia-modal en materias.templ)
	if strings.Contains(r.Header.Get("Content-Type"), "application/x-www-form-urlencoded") {
		idUsuario, err := ObtenerUsuarioIDRedis(r.Context(), r, rdb)
		if err != nil {
			RedirigirSinSesion(w, r)
			return
		}

		if err := r.ParseForm(); err != nil {
			http.Error(w, "Error al procesar el formulario de materia", http.StatusBadRequest)
			return
		}

		titulo := strings.TrimSpace(r.FormValue("titulo"))
		if titulo == "" {
			http.Error(w, "El nombre de la materia es obligatorio", http.StatusBadRequest)
			return
		}

		_, err = q.CreateMateria(r.Context(), db.CreateMateriaParams{
			IDUsuario: idUsuario,
			Titulo:    titulo,
		})
		if err != nil {
			log.Printf("ERROR BD al crear materia para usuario #%d: %v", idUsuario, err)
			http.Error(w, "Error al crear la materia", http.StatusInternalServerError)
			return
		}

		renderMateriasActualizadas(w, r, q, rdb, idUsuario)
		return
	}

	// 2. Caso petición JSON (para API REST y pruebas automatizadas)
	err := json.NewDecoder(r.Body).Decode(&newMateria)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if strings.TrimSpace(newMateria.Titulo) == "" || newMateria.IDUsuario <= 0 {
		http.Error(w, "Todos los campos son obligatorios y no pueden contener solo espacios", http.StatusBadRequest)
		return
	}

	materiaCreada, err := q.CreateMateria(r.Context(), newMateria)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(materiaCreada)
}

func GetListMateriasByUsuario(w http.ResponseWriter, r *http.Request, q *db.Queries) {
	idStr := r.PathValue("id")

	id64, err := strconv.ParseInt(idStr, 10, 32)
	if err != nil {
		http.Error(w, "ID de usuario inválido", http.StatusBadRequest)
		return
	}

	id := int32(id64)

	listaMateria, err := q.ListMateriasByUsuario(r.Context(), id)
	if err != nil {
		http.Error(w, "Usuario para listar materias no encontrado", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(listaMateria)
}

func PutUpdateMateria(w http.ResponseWriter, r *http.Request, q *db.Queries) {
	var MateriaRecibida db.UpdateMateriaParams

	err := json.NewDecoder(r.Body).Decode(&MateriaRecibida)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if strings.TrimSpace(MateriaRecibida.Titulo) == "" ||
		MateriaRecibida.IDLista <= 0 {
		http.Error(w, "Todos los campos son obligatorios y no pueden contener solo espacios", http.StatusBadRequest)
		return
	}

	MateriaActualizada, err := q.UpdateMateria(r.Context(), MateriaRecibida)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(MateriaActualizada)
}

func DeleteMateria(w http.ResponseWriter, r *http.Request, q *db.Queries) {
	idStr := r.PathValue("id")

	parsed, err := strconv.ParseInt(idStr, 10, 32)
	if err != nil {
		http.Error(w, "El ID debe ser un número entero de 32 bits válido", http.StatusBadRequest)
		return
	}

	idMateria := int32(parsed)

	err = q.DeleteMateria(r.Context(), idMateria)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// ==========================================
// CONTROLADORES DE VISTAS (MATERIAS + HTMX)
// ==========================================

// cargarMateriasConTareas consulta todas las materias del usuario y sus respectivas
// tareas para armar el tablero Kanban de la vista materias.templ.
func cargarMateriasConTareas(ctx context.Context, q *db.Queries, idUsuario int32) ([]views.MateriaConTareas, error) {
	materias, err := q.ListMateriasByUsuario(ctx, idUsuario)
	if err != nil {
		return nil, err
	}

	resultado := make([]views.MateriaConTareas, 0, len(materias))
	for _, m := range materias {
		tareas, err := q.ListTareasByMateria(ctx, m.IDLista)
		if err != nil {
			return nil, err
		}
		resultado = append(resultado, views.MateriaConTareas{
			Materia: m,
			Tareas:  tareas,
		})
	}

	return resultado, nil
}

// materiaPerteneceAUsuario verifica que una materia (id_lista) exista y pertenezca al usuario activo.
func materiaPerteneceAUsuario(ctx context.Context, q *db.Queries, idLista int32, idUsuario int32) (bool, error) {
	materias, err := q.ListMateriasByUsuario(ctx, idUsuario)
	if err != nil {
		return false, err
	}
	for _, m := range materias {
		if m.IDLista == idLista {
			return true, nil
		}
	}
	return false, nil
}

// renderMateriasActualizadas recarga el tablero del usuario y responde con el fragmento
// HTMX (MateriasContent) o la página completa (MateriasPage) según el header HX-Request.
func renderMateriasActualizadas(w http.ResponseWriter, r *http.Request, q *db.Queries, rdb *redis.Client, idUsuario int32) {
	materiasConTareas, err := cargarMateriasConTareas(r.Context(), q, idUsuario)
	if err != nil {
		log.Printf("ERROR BD al cargar materias del usuario #%d: %v", idUsuario, err)
		http.Error(w, "Error al cargar las materias", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if r.Header.Get("HX-Request") == "true" {
		views.MateriasContent(materiasConTareas).Render(r.Context(), w)
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

	views.MateriasPage(usuario, materiasConTareas).Render(r.Context(), w)
}

// MateriasView renderiza el tablero de materias y tareas del usuario activo en Redis.
// Soporta navegación interna con HTMX (MateriasContent) y carga completa / F5 (MateriasPage).
func MateriasView(w http.ResponseWriter, r *http.Request, q *db.Queries, rdb *redis.Client) {
	idUsuario, err := ObtenerUsuarioIDRedis(r.Context(), r, rdb)
	if err != nil {
		RedirigirSinSesion(w, r)
		return
	}

	renderMateriasActualizadas(w, r, q, rdb, idUsuario)
}
