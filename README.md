# `go_tasks` — Gestor Académico de Materias, Tareas y Cronómetro

Aplicación web desarrollada en **Go** con renderizado SSR mediante **Templ**, interactividad con **HTMX**, base de datos **PostgreSQL** (`sqlc` + `pgx/v5`), manejo de sesiones con **Redis** (`go-redis/v9`) y estilos CSS nativos (sin archivos `.js`).

## Documentación del Proyecto

Toda la especificación de arquitectura, base de datos, diagrama relacional (ERD), consultas `sqlc`, rutas/handlers, vistas Templ y reglas de negocio se encuentra en:

- [Documentación Técnica y de Dominio](docs/documentacion_proyecto.md)

---

## Requisitos Previos

Para ejecutar el proyecto localmente se requiere tener instalados:
- **Go** (1.26+)
- **Docker** y **Docker Compose** (para levantar PostgreSQL y Redis)
- **sqlc** (generación de código SQL en `db/`)
- **templ** (compilación de vistas `views/*.templ`)

---

## Ejecución de la Aplicación

El proyecto cuenta con un `Makefile` y el script [servidor.sh](servidor.sh) en la raíz para automatizar todo el ciclo de ejecución.

### 1. Levantar la aplicación (`make start`)

```bash
make start
```

También puede ejecutarse directamente con `./servidor.sh start` (asegurando permisos de ejecución con `chmod +x servidor.sh`).

Este comando realiza automáticamente:
1. **Tareas previas:**
   - Ejecuta `sqlc generate` para actualizar el paquete `db/`.
   - Ejecuta `templ generate` para compilar las vistas `views/*.templ`.
   - Compila el binario del servidor en `bin/server`.
   - Limpia contenedores y volúmenes previos (`docker compose down -v --remove-orphans`).
   - Levanta los contenedores de **PostgreSQL** (`5432`) y **Redis** (`6379`) y espera hasta que ambos respondan correctamente.
2. **Ejecución de la aplicación:**
   - Inicia el servidor HTTP en segundo plano escuchando en **`http://localhost:8080`**.

### 2. Detener la aplicación y limpiar el entorno (`make stop`)

```bash
make stop
```

También puede ejecutarse directamente con `./servidor.sh stop`.

Realiza las **tareas posteriores**:
- Detiene el proceso del servidor Go en ejecución.
- Ejecuta `docker compose down -v --remove-orphans` para detener y eliminar los contenedores y volúmenes de PostgreSQL y Redis.

### 3. Comandos adicionales disponibles

- `make restart` — Detiene el servidor y contenedores (`stop`) y vuelve a compilar y levantar todo limpio (`start`).
- `make generate` — Ejecuta únicamente `sqlc generate`.
- `make templ` — Ejecuta únicamente `templ generate`.
