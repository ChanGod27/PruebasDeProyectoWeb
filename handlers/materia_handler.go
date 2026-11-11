package handlers

import (
	"ServidorEntregable/db"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
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