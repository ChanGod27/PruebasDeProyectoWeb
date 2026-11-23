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

func CreateMateria(w http.ResponseWriter, r *http.Request, q *db.Queries) {
	var newMateria db.CreateMateriaParams

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

// MateriasView renderiza el tablero de materias y tareas del usuario activo en Redis.
// Soporta navegación interna con HTMX (MateriasContent) y carga completa / F5 (MateriasPage).
func MateriasView(w http.ResponseWriter, r *http.Request, q *db.Queries, rdb *redis.Client) {
	// 1. Verificar sesión activa en Redis.
	idUsuario, err := ObtenerUsuarioIDRedis(r.Context(), r, rdb)
	if err != nil {
		RedirigirSinSesion(w, r)
		return
	}

	// 2. Cargar las materias del usuario junto con sus tareas pendientes y completadas.
	materiasConTareas, err := cargarMateriasConTareas(r.Context(), q, idUsuario)
	if err != nil {
		log.Printf("ERROR BD al cargar materias del usuario #%d: %v", idUsuario, err)
		http.Error(w, "Error al cargar las materias", http.StatusInternalServerError)
		return
	}

	// 3. Si la petición vino por HTMX, devolver solo el fragmento interno (#main-content).
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if r.Header.Get("HX-Request") == "true" {
		views.MateriasContent(materiasConTareas).Render(r.Context(), w)
		return
	}

	// 4. Si es carga directa o F5, buscar los datos del usuario para el header de LayoutBase.
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
