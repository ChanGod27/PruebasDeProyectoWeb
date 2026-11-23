package handlers

import (
	"ServidorEntregable/db"
	"ServidorEntregable/views"
	"context"
	"log"
	"net/http"
	"strconv"
	"strings"

	"github.com/redis/go-redis/v9"
)

// ==========================================
// OPERACIONES DE MATERIA
// ==========================================

// operacion principal de creacion de materia
func CreateMateria(w http.ResponseWriter, r *http.Request, q *db.Queries, rdb *redis.Client) {
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
}

// operacion de actualizacion de materia
func PutUpdateMateria(w http.ResponseWriter, r *http.Request, q *db.Queries, rdb *redis.Client) {
	idUsuario, err := ObtenerUsuarioIDRedis(r.Context(), r, rdb)
	if err != nil {
		RedirigirSinSesion(w, r)
		return
	}

	parsed, err := strconv.ParseInt(r.PathValue("id"), 10, 32)
	if err != nil || parsed <= 0 {
		http.Error(w, "El ID de materia debe ser válido", http.StatusBadRequest)
		return
	}
	idMateria := int32(parsed)

	pertenece, err := materiaPerteneceAUsuario(r.Context(), q, idMateria, idUsuario)
	if err != nil {
		log.Printf("ERROR BD al validar materia #%d: %v", idMateria, err)
		http.Error(w, "Error al validar permisos de la materia", http.StatusInternalServerError)
		return
	}
	if !pertenece {
		http.Error(w, "No tienes permiso para editar esta materia", http.StatusForbidden)
		return
	}

	if err := r.ParseForm(); err != nil {
		http.Error(w, "Error al procesar el formulario de edición", http.StatusBadRequest)
		return
	}

	titulo := strings.TrimSpace(r.FormValue("titulo"))
	if titulo == "" {
		http.Error(w, "El nombre de la materia es obligatorio", http.StatusBadRequest)
		return
	}

	_, err = q.UpdateMateria(r.Context(), db.UpdateMateriaParams{
		IDLista: idMateria,
		Titulo:  titulo,
	})
	if err != nil {
		log.Printf("ERROR BD al actualizar materia #%d: %v", idMateria, err)
		http.Error(w, "Error al actualizar la materia", http.StatusInternalServerError)
		return
	}

	renderMateriasActualizadas(w, r, q, rdb, idUsuario)
}

// operacion de eliminacion de materia
func DeleteMateria(w http.ResponseWriter, r *http.Request, q *db.Queries, rdb *redis.Client) {
	idStr := r.PathValue("id")

	parsed, err := strconv.ParseInt(idStr, 10, 32)
	if err != nil || parsed <= 0 {
		http.Error(w, "El ID debe ser un número entero de 32 bits válido", http.StatusBadRequest)
		return
	}

	idMateria := int32(parsed)

	idUsuario, err := ObtenerUsuarioIDRedis(r.Context(), r, rdb)
	if err != nil {
		RedirigirSinSesion(w, r)
		return
	}

	pertenece, err := materiaPerteneceAUsuario(r.Context(), q, idMateria, idUsuario)
	if err != nil {
		log.Printf("ERROR BD al validar materia #%d: %v", idMateria, err)
		http.Error(w, "Error al validar permisos de la materia", http.StatusInternalServerError)
		return
	}
	if !pertenece {
		http.Error(w, "No tienes permiso para eliminar esta materia", http.StatusForbidden)
		return
	}

	if err := q.DeleteMateria(r.Context(), idMateria); err != nil {
		log.Printf("ERROR BD al eliminar materia #%d: %v", idMateria, err)
		http.Error(w, "Error al eliminar la materia", http.StatusInternalServerError)
		return
	}

	renderMateriasActualizadas(w, r, q, rdb, idUsuario)
}

// se obtiene todas las materias de un usuario y se les carga sus tareas
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

// se verifica que una materia exista y pertenezca al usuario
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

// se recarga el tablero del usuario con sus materias actualizadas
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

// serenderiza el tablero de materias y tareas del usuario
func MateriasView(w http.ResponseWriter, r *http.Request, q *db.Queries, rdb *redis.Client) {
	idUsuario, err := ObtenerUsuarioIDRedis(r.Context(), r, rdb)
	if err != nil {
		RedirigirSinSesion(w, r)
		return
	}

	renderMateriasActualizadas(w, r, q, rdb, idUsuario)
}
