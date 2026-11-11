-- 1. Tabla Usuario
CREATE TABLE usuario (
    id_usuario SERIAL PRIMARY KEY,
    nombre VARCHAR(100) NOT NULL,
    apellido VARCHAR(100) NOT NULL,
    email VARCHAR(255) NOT NULL UNIQUE,
    contrasena VARCHAR(255) NOT NULL
);

-- 2. Tabla Lista de Tareas
CREATE TABLE lista_tarea (
    id_lista SERIAL PRIMARY KEY,
    id_usuario INT NOT NULL,
    titulo VARCHAR(150) NOT NULL,
    CONSTRAINT fk_lista_usuario 
        FOREIGN KEY (id_usuario) 
        REFERENCES usuario(id_usuario) 
        ON DELETE CASCADE
);

-- 3. Tabla Tarea
CREATE TABLE tarea (
    id_tarea SERIAL PRIMARY KEY,
    id_lista INT NOT NULL,
    titulo VARCHAR(150) NOT NULL,
    fecha_inicio TIMESTAMP NULL,
    fecha_finalizacion TIMESTAMP,
    descripcion TEXT,
    completada BOOLEAN NOT NULL DEFAULT FALSE,
    CONSTRAINT fk_tarea_lista 
        FOREIGN KEY (id_lista) 
        REFERENCES lista_tarea(id_lista) 
        ON DELETE CASCADE
);

-- 4. Tabla Horas Trabajadas (Sub-entidad de Tarea)
CREATE TABLE horas_trabajadas (
    id_horas SERIAL PRIMARY KEY,
    id_tarea INT NOT NULL,
    timestamp_inicio TIMESTAMP NOT NULL,
    timestamp_fin TIMESTAMP,
    CONSTRAINT fk_horas_tarea 
        FOREIGN KEY (id_tarea) 
        REFERENCES tarea(id_tarea)
        ON DELETE CASCADE
);