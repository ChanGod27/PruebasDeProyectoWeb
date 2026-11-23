# `go_tasks` — Gestor Académico de Materias, Tareas y Cronómetro

Aplicación web desarrollada en **Go**

## Documentación del Proyecto

Toda la especificación de arquitectura, base de datos, diagrama relacional (ERD), consultas `sqlc`, rutas/handlers, vistas Templ y reglas de negocio se encuentra en:

- [Documentación Técnica y de Dominio](docs/documentacion_proyecto.md)
---

## Ejecución de la Aplicación


### 1. Levantar la aplicación (`make start`)

```bash
make start
```

También puede ejecutarse directamente con `./servidor.sh start` (asegurando permisos de ejecución con `chmod +x servidor.sh`).

### 2. Detener la aplicación y limpiar el entorno (`make stop`)

```bash
make stop
```

También puede ejecutarse directamente con `./servidor.sh stop`.

### 3. Comandos adicionales disponibles

- `make restart` — Detiene el servidor y contenedores (`stop`) y vuelve a compilar y levantar todo limpio (`start`).
- `make generate` — Ejecuta únicamente `sqlc generate`.
- `make templ` — Ejecuta únicamente `templ generate`.
