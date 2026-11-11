/* =============================================================================
   1. USUARIO Y AUTENTICACIÓN
   ============================================================================= */

-- name: CreateUser :one
INSERT INTO usuario (
    nombre,
    apellido,
    email,
    contrasena
) VALUES (
    $1, $2, $3, $4
)
RETURNING id_usuario, nombre, apellido, email;

-- name: GetUserByEmail :one
SELECT id_usuario, nombre, apellido, email, contrasena
FROM usuario
WHERE email = $1;

-- name: GetUserByID :one
SELECT id_usuario, nombre, apellido, email
FROM usuario
WHERE id_usuario = $1;

-- name: ListUsers :many
SELECT id_usuario, nombre, apellido, email, contrasena
FROM usuario
ORDER BY id_usuario ASC;


/* =============================================================================
   2. MATERIAS (Lista de Tarea)
   ============================================================================= */

-- name: CreateMateria :one
INSERT INTO lista_tarea (
    id_usuario,
    titulo
) VALUES (
    $1, $2
)
RETURNING id_lista, id_usuario, titulo;

-- name: ListMateriasByUsuario :many
SELECT id_lista, id_usuario, titulo
FROM lista_tarea
WHERE id_usuario = $1
ORDER BY id_lista ASC;

-- name: UpdateMateria :one
UPDATE lista_tarea
SET 
    titulo = $2
WHERE id_lista = $1
RETURNING id_lista, id_usuario, titulo;

-- name: DeleteMateria :exec
DELETE FROM lista_tarea
WHERE id_lista = $1;


/* =============================================================================
   3. TAREAS
   ============================================================================= */

-- name: CreateTarea :one
INSERT INTO tarea (
    id_lista,
    titulo,
    fecha_inicio,
    fecha_finalizacion,
    descripcion,
    completada
) VALUES (
    $1, $2, $3, $4, $5, FALSE
)
RETURNING id_tarea, id_lista, titulo, fecha_inicio, fecha_finalizacion, descripcion, completada;

-- name: GetTareaByID :one
SELECT id_tarea, id_lista, titulo, fecha_inicio, fecha_finalizacion, descripcion, completada
FROM tarea
WHERE id_tarea = $1;

-- name: ListTareasByMateria :many
-- Para cuando abrís las columnas en el tablero
SELECT id_tarea, id_lista, titulo, fecha_inicio, fecha_finalizacion, descripcion, completada
FROM tarea
WHERE id_lista = $1
ORDER BY id_tarea ASC;

-- name: ListTareasByRangoFechas :many
-- Para pintar el calendario entre dos fechas
SELECT id_tarea, id_lista, titulo, fecha_inicio, fecha_finalizacion, descripcion, completada
FROM tarea
WHERE fecha_finalizacion >= $1 AND fecha_finalizacion <= $2
ORDER BY fecha_finalizacion ASC;

-- name: UpdateTarea :one
-- Update común y corriente: le pasás todos los valores actualizados
UPDATE tarea
SET 
    titulo = $2,
    fecha_inicio = $3,
    fecha_finalizacion = $4,
    descripcion = $5,
    completada = $6
WHERE id_tarea = $1
RETURNING id_tarea, id_lista, titulo, fecha_inicio, fecha_finalizacion, descripcion, completada;

-- name: DeleteTarea :exec
DELETE FROM tarea
WHERE id_tarea = $1;


/* =============================================================================
   4. CRONÓMETRO Y HORAS TRABAJADAS
   ============================================================================= */

-- name: CreateHorasTrabajadas :one
INSERT INTO horas_trabajadas (
    id_tarea,
    timestamp_inicio,
    timestamp_fin
) VALUES (
    $1, $2, $3
)
RETURNING id_horas, id_tarea, timestamp_inicio, timestamp_fin;

-- name: ListHorasByTarea :many
-- Ver sesiones de una tarea específica
SELECT id_horas, id_tarea, timestamp_inicio, timestamp_fin
FROM horas_trabajadas
WHERE id_tarea = $1
ORDER BY timestamp_inicio DESC;

-- name: ListHorasByRangoFechas :many
-- Para estadísticas o bloques en calendario
SELECT id_horas, id_tarea, timestamp_inicio, timestamp_fin
FROM horas_trabajadas
WHERE timestamp_inicio >= $1 AND timestamp_inicio <= $2
ORDER BY timestamp_inicio DESC;

-- name: UpdateHorasTrabajadas :one
UPDATE horas_trabajadas
SET 
    timestamp_inicio = $2,
    timestamp_fin = $3
WHERE id_horas = $1
RETURNING id_horas, id_tarea, timestamp_inicio, timestamp_fin;

-- name: DeleteHorasTrabajadas :exec
DELETE FROM horas_trabajadas
WHERE id_horas = $1;