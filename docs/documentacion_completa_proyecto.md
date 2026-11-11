# Documentación Integral del Proyecto — `go_tasks` (`ServidorEntregable`)

Esta guía documenta **todo el proyecto** de principio a fin: su arquitectura, el propósito de cada carpeta, el funcionamiento detallado de **todos los archivos** que componen el repositorio y la explicación línea por línea de los bloques de código fundamentales.

---

## Tabla de Contenidos

1. [Visión General y Arquitectura](#1-visión-general-y-arquitectura)
2. [Estructura Completa de Directorios y Archivos](#2-estructura-completa-de-directorios-y-archivos)
3. [Archivos de Raíz y Configuración](#3-archivos-de-raíz-y-configuración)
   - [`go.mod` y `go.sum`](#31-gomod-y-gosum)
   - [`docker-compose.yml`](#32-docker-composeyml)
   - [`sqlc.yaml`](#33-sqlcyaml)
   - [`Makefile`](#34-makefile)
   - [`README.md`](#35-readmemd)
   - [`main.go` (Punto de entrada y Enrutador HTTP)](#36-maingo-punto-de-entrada-y-enrutador-http)
4. [Capa de Base de Datos y Persistencia (`db/`)](#4-capa-de-base-de-datos-y-persistencia-db)
   - [`db/schema.sql` (Modelo Relacional DDL)](#41-dbschemasql-modelo-relacional-ddl)
   - [`db/queries.sql` (Consultas SQL parametrizadas para sqlc)](#42-dbqueriessql-consultas-sql-parametrizadas-para-sqlc)
   - [`db/coneccion_db.go` (Conexión con PostgreSQL vía pgx)](#43-dbconeccion_dbgo-conexión-con-postgresql-vía-pgx)
   - [`db/db.go` (Interfaz DBTX e inicializador generado por sqlc)](#44-dbdbgo-interfaz-dbtx-e-inicializador-generado-por-sqlc)
   - [`db/models.go` (Estructuras de dominio generadas por sqlc)](#45-dbmodelsgo-estructuras-de-dominio-generadas-por-sqlc)
   - [`db/queries.sql.go` (Métodos CRUD en Go generados por sqlc)](#46-dbqueriessqlgo-métodos-crud-en-go-generados-por-sqlc)
   - [`db/db_test.go` (Pruebas de Integración de Base de Datos)](#47-dbdb_testgo-pruebas-de-integración-de-base-de-datos)
5. [Capa de Controladores HTTP (`handlers/`)](#5-capa-de-controladores-http-handlers)
   - [`handlers/usuario_handler.go` (API de Usuarios y Vistas Templ)](#51-handlersusuario_handlergo-api-de-usuarios-y-vistas-templ)
   - [`handlers/materia_handler.go` (API CRUD de Materias / Listas)](#52-handlersmateria_handlergo-api-crud-de-materias--listas)
   - [`handlers/tarea_handler.go` (API CRUD de Tareas y Rango de Fechas)](#53-handlerstarea_handlergo-api-crud-de-tareas-y-rango-de-fechas)
   - [`handlers/horas_trabajadas_handler.go` (API CRUD de Sesiones de Estudio)](#54-handlershoras_trabajadas_handlergo-api-crud-de-sesiones-de-estudio)
6. [Capa de Vistas Server-Side Rendering (`view/`)](#6-capa-de-vistas-server-side-rendering-view)
   - [`view/layout.templ` y `view/layout_templ.go`](#61-viewlayouttempl-y-viewlayout_templgo)
   - [`view/usuario.templ` y `view/usuario_templ.go`](#62-viewusuariotempl-y-viewusuario_templgo)
   - [`view/create_usuario.templ` y `view/create_usuario_templ.go`](#63-viewcreate_usuariotempl-y-viewcreate_usuario_templgo)
7. [Capa de Recursos Estáticos y Maquetas (`static/`)](#7-capa-de-recursos-estáticos-y-maquetas-static)
   - [`static/css/style.css`](#71-staticcssstylecss)
   - [`static/index.html`, `static/usuarios.html`, `static/calendario.html`, `static/materias.html`](#72-maquetas-html-estáticas-indexhtml-usuarioshtml-calendariohtml-materiashtml)
8. [Automatización, Scripts y Testing End-to-End (`tests/`)](#8-automatización-scripts-y-testing-end-to-end-tests)
   - [`tests/server.sh`, `tests/start.sh` y `tests/stop.sh`](#81-testsserversh-testsstartsh-y-testsstopsh)
   - [`tests/test.sh`](#82-teststestsh)
   - [`tests/request.hurl`](#83-testsrequesthurl)
9. [Documentación Complementaria (`docs/`) y Binarios (`bin/`)](#9-documentación-complementaria-docs-y-binarios-bin)

---

## 1. Visión General y Arquitectura

**`go_tasks`** (módulo Go `ServidorEntregable`) es una aplicación web y servidor API REST para la gestión académica y productiva de estudiantes y profesionales. Permite administrar:
1. **Usuarios (`usuario`)**: Cuentas personales e inicio de sesión.
2. **Materias / Listas de Tareas (`lista_tarea`)**: Agrupadores temáticos o columnas de tablero que pertenecen a un usuario.
3. **Tareas (`tarea`)**: Actividades con descripción, estado de completitud, fecha de inicio y fecha límite de finalización.
4. **Horas Trabajadas (`horas_trabajadas`)**: Bloques de tiempo registrados (inicio y fin) asociados a una tarea específica.

### Flujo Arquitectónico

```mermaid
flowchart TD
    Client["Navegador Web / Cliente HTTP (Hurl)"]
    Router["main.go (net/http ServeMux)"]
    Static["static/ (CSS y HTML estático)"]
    Handlers["handlers/*_handler.go"]
    Views["view/*.templ -> *_templ.go"]
    SQLC["db/queries.sql.go (Generado por sqlc)"]
    PGX["db/coneccion_db.go (Driver pgx/v5)"]
    Postgres[("PostgreSQL 18 (Docker)")]

    Client -->|Petición HTTP| Router
    Router -->|/css/* o /static/*| Static
    Router -->|Rutas Web / API| Handlers
    Handlers -->|Renderizado HTML SSR| Views
    Handlers -->|Operaciones de Datos| SQLC
    SQLC -->|Consultas Tipadas| PGX
    PGX -->|TCP :5432| Postgres
```

---

## 2. Estructura Completa de Directorios y Archivos

```text
PruebasDeProyectoWeb/
├── main.go                             # Punto de entrada de la aplicación y configuración de rutas HTTP
├── go.mod                              # Definición del módulo Go y dependencias directas/indirectas
├── go.sum                              # Sumas criptográficas de verificación de dependencias
├── sqlc.yaml                           # Configuración del generador de código SQL a Go (sqlc)
├── docker-compose.yml                  # Definición del contenedor PostgreSQL 18 y su inicialización
├── Makefile                            # Comandos abreviados para ciclo de vida y pruebas
├── README.md                           # Guía rápida de inicio y enlaces de documentación
├── server.log                          # Registro de salida estándar y errores del servidor en segundo plano
├── bin/
│   ├── server                          # Binario ejecutable compilado del servidor Go
│   └── server.pid                      # Archivo temporal que almacena el PID del proceso en ejecución
├── db/
│   ├── schema.sql                      # Definición DDL de las 4 tablas relacionales y claves foráneas
│   ├── queries.sql                     # Consultas SQL anotadas para que sqlc genere funciones Go
│   ├── coneccion_db.go                 # Función AbridDB para abrir y verificar la conexión con PostgreSQL
│   ├── db.go                           # (Generado por sqlc) Interfaz DBTX y constructor New(db)
│   ├── models.go                       # (Generado por sqlc) Structs Go de las tablas SQL
│   ├── queries.sql.go                  # (Generado por sqlc) Implementación en Go de cada consulta SQL
│   └── db_test.go                      # Suite de tests de integración para validar todas las consultas CRUD
├── handlers/
│   ├── usuario_handler.go              # Controladores HTTP (API JSON + Vistas HTML Templ) de Usuarios y Login
│   ├── materia_handler.go              # Controladores HTTP REST JSON para Materias (ListaTarea)
│   ├── tarea_handler.go                # Controladores HTTP REST JSON para Tareas
│   └── horas_trabajadas_handler.go     # Controladores HTTP REST JSON para Horas Trabajadas
├── view/
│   ├── layout.templ                    # Plantilla Templ con el LayoutBase HTML y la página principal
│   ├── layout_templ.go                 # (Generado por templ) Código Go compilado de layout.templ
│   ├── usuario.templ                   # Plantilla Templ con ListaUsuarios, LoginPage y TP6Aviso
│   ├── usuario_templ.go                # (Generado por templ) Código Go compilado de usuario.templ
│   ├── create_usuario.templ            # Plantilla Templ con el formulario HTML para registrar usuarios
│   └── create_usuario_templ.go         # (Generado por templ) Código Go compilado de create_usuario.templ
├── static/
│   ├── css/
│   │   └── style.css                   # Hoja de estilos CSS nativa (Dark Mode, variables, componentes)
│   ├── index.html                      # Maqueta estática de inicio de sesión
│   ├── usuarios.html                   # Maqueta estática de selección y alta de usuarios
│   ├── calendario.html                 # Maqueta estática con aviso de entrega TP6 (Calendario)
│   └── materias.html                   # Maqueta estática con aviso de entrega TP6 (Materias)
├── tests/
│   ├── server.sh                       # Script principal de gestión del servidor (start, stop, status, logs)
│   ├── start.sh                        # Atajo ejecutable hacia `./tests/server.sh start`
│   ├── stop.sh                         # Atajo ejecutable hacia `./tests/server.sh stop`
│   ├── test.sh                         # Orquestador completo de pruebas (Docker + go test + Hurl)
│   └── request.hurl                    # Suite de pruebas de integración HTTP sobre todos los endpoints REST
└── docs/
    ├── documentacion_dominio.md        # Especificación del modelo de dominio y reglas de negocio
    ├── documentacion_vistas_templ.md   # Guía específica del motor de plantillas Templ
    └── documentacion_completa_proyecto.md # Este documento maestro
```

---

## 3. Archivos de Raíz y Configuración

### 3.1. `go.mod` y `go.sum`
- **Propósito**: Definen el módulo de Go llamado `ServidorEntregable`, la versión del compilador (`go 1.26.6`) y los paquetes externos utilizados.
- **Dependencias clave**:
  - `github.com/jackc/pgx/v5`: Driver nativo y de alto rendimiento para PostgreSQL.
  - `github.com/a-h/templ`: Motor de plantillas HTML tipadas para Go.

```go
module ServidorEntregable

go 1.26.6

require github.com/jackc/pgx/v5 v5.11.0

require (
	github.com/a-h/templ v0.3.1100 // indirect
	github.com/jackc/pgpassfile v1.0.0 // indirect
	github.com/jackc/pgservicefile v0.0.0-20240606120523-5a60cdf6a761 // indirect
	golang.org/x/text v0.29.0 // indirect
)
```

---

### 3.2. `docker-compose.yml`
- **Propósito**: Levanta un contenedor ligero de PostgreSQL (`postgres:18-alpine`) preconfigurado con credenciales y carga automáticamente el esquema de tablas al inicializarse por primera vez.

```yaml
services:
  db:
    image: postgres:18-alpine
    environment:
      POSTGRES_DB: go_tasks_db
      POSTGRES_USER: trabajo_web
      POSTGRES_PASSWORD: carlos-luca-hermes
    ports:
      - "5432:5432"
    volumes:
      - ./db/schema.sql:/docker-entrypoint-initdb.d/schema.sql
    healthcheck:
      test: [ "CMD-SHELL", "pg_isready -U postgres -d mi_dominio_db" ]
      interval: 2s
      timeout: 5s
      retries: 5
```
- **Explicación**:
  - `ports: - "5432:5432"`: Expone el puerto de PostgreSQL al host local para que la aplicación Go se conecte a `localhost:5432`.
  - `volumes`: Monta `./db/schema.sql` dentro de `/docker-entrypoint-initdb.d/`, haciendo que PostgreSQL cree las 4 tablas automáticamente cuando se crea el volumen.
  - `healthcheck`: Define una prueba periódica cada 2 segundos usando `pg_isready` para saber cuándo el motor de base de datos está listo para recibir conexiones.

---

### 3.3. `sqlc.yaml`
- **Propósito**: Configura la herramienta `sqlc`, que lee archivos `.sql` y genera código Go fuertemente tipado sin necesidad de usar un ORM pesado.

```yaml
version: "2"
sql:
  - engine: "postgresql"
    schema: "db/schema.sql"      # Ubicación de las tablas/DDL
    queries: "db/queries.sql"    # Ubicación de las consultas con anotaciones sqlc
    gen:
      go:
        package: "db"            # Nombre del package de Go para los archivos generados
        out: "db"                # Directorio donde sqlc escribirá el código Go
        sql_package: "pgx/v5"    # Indica que use los tipos y conexiones de pgx/v5
```

---

### 3.4. `Makefile`
- **Propósito**: Provee atajos de terminal estandarizados para compilar, generar código, correr pruebas y administrar el ciclo de vida del servidor y de Docker.

```makefile
.PHONY: test clean generate start stop stop-all restart status logs

test:
	./tests/test.sh

clean:
	docker compose down -v --remove-orphans

generate:
	sqlc generate

start: generate
	./tests/server.sh start
```
- **Explicación**:
  - `make start` depende primero de `generate` (`sqlc generate`), garantizando que los archivos autogenerados de la capa `db/` existan antes de compilar y arrancar el servidor con `./tests/server.sh start`.
  - `make test` ejecuta el pipeline completo de pruebas automatizadas.
  - `make clean` destruye los contenedores y volúmenes (`-v`) para dejar el entorno desde cero.

---

### 3.5. `README.md`
- **Propósito**: Archivo de presentación del repositorio con enlaces directos a los documentos de dominio y vistas, además de las instrucciones básicas para ejecutar los tests.

---

### 3.6. `main.go` (Punto de entrada y Enrutador HTTP)
- **Propósito**: Es el corazón del servidor ejecutable. Se encarga de:
  1. Abrir la conexión hacia PostgreSQL.
  2. Instanciar el repositorio de consultas (`queries := db.New(conn)`).
  3. Configurar el servidor de archivos estáticos (`/css/` y `/static/`).
  4. Registrar todos los endpoints de vistas HTML (SSR con Templ) y de la API REST JSON utilizando el enrutador estándar de Go 1.22+ (`http.HandleFunc("METODO /ruta/{param}", ...)`).
  5. Iniciar el servidor HTTP en el puerto `:8080`.

#### Explicación de bloques de código de `main.go`:

**1. Conexión a la Base de Datos e Inicialización de `sqlc`:**
```go
func main() {
	ctx := context.Background()
	connStr := "postgres://trabajo_web:carlos-luca-hermes@localhost:5432/go_tasks_db?sslmode=disable"

	conn, errOpen := db.AbridDB(ctx, connStr)
	if errOpen != nil {
		fmt.Printf("error crítico al conectar a la base de datos: %v", errOpen)
		return
	}
	defer conn.Close(ctx)

	// Instanciar objeto de la base de datos (userRepository)
	queries := db.New(conn)
```
- Se crea un contexto base (`context.Background()`), se conecta a PostgreSQL mediante `db.AbridDB` y se agenda el cierre limpio de la conexión con `defer conn.Close(ctx)`.
- `db.New(conn)` envuelve la conexión en una estructura `*db.Queries` que se inyecta por parámetro a todos los handlers.

**2. Servidor de Archivos Estáticos:**
```go
	staticDir := "./static"
	http.Handle("/css/", http.StripPrefix("/css/", http.FileServer(http.Dir("./static/css"))))
	http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir(staticDir))))
```
- Permite que las plantillas HTML carguen `/css/style.css` directamente desde la carpeta local `./static/css`, y que cualquier archivo en `./static` sea accesible bajo `/static/`.

**3. Rutas de Vistas HTML (Templ) y Rutas API REST:**
```go
	// Página principal: Lista de usuarios (layout.templ)
	http.HandleFunc("GET /{$}", func(w http.ResponseWriter, r *http.Request) {
		handlers.UsuariosView(w, r, queries)
	})
	http.HandleFunc("GET /usuarios", func(w http.ResponseWriter, r *http.Request) {
		handlers.UsuariosView(w, r, queries)
	})
```
- El patrón `"GET /{$}"` (introducido en el router nativo de Go 1.22) coincide **únicamente** con la raíz exacta `/`, evitando capturar rutas inexistentes.
- A continuación, se registran las rutas de vistas (`GET /usuarios/nuevo`, `GET /login`, `POST /login`, `GET /tp6`, `GET /calendario`, `GET /materias`) y todas las rutas CRUD para `/usuarios`, `/materias`, `/tareas` y `/horas`, cerrando con `http.ListenAndServe(":8080", nil)`.

---

## 4. Capa de Base de Datos y Persistencia (`db/`)

### 4.1. `db/schema.sql` (Modelo Relacional DDL)
- **Propósito**: Define la estructura física de las 4 tablas en PostgreSQL y sus restricciones de integridad referencial (`FOREIGN KEY ... ON DELETE CASCADE`).

```sql
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
```
- **Detalles clave**:
  - `email VARCHAR(255) NOT NULL UNIQUE`: Impide a nivel de motor de base de datos que dos usuarios se registren con el mismo correo electrónico.
  - `ON DELETE CASCADE`: Garantiza que al borrar un registro padre (ej. una materia en `lista_tarea`), PostgreSQL elimine automáticamente sus registros hijos (`tarea` y `horas_trabajadas`) sin dejar datos huérfanos.

---

### 4.2. `db/queries.sql` (Consultas SQL parametrizadas para sqlc)
- **Propósito**: Contiene las 19 consultas SQL del sistema, precedidas por comentarios especiales (`-- name: NombreFuncion :tipoRetorno`) que `sqlc` interpreta para generar funciones Go.
- **Tipos de anotación usados**:
  - `:one`: La consulta retorna exactamente 1 fila (ej. `INSERT ... RETURNING` o `SELECT ... WHERE id = $1`).
  - `:many`: La consulta retorna un slice de filas (`[]Modelo`).
  - `:exec`: Ejecuta un comando sin retornar filas (ej. `DELETE`).

**Ejemplo de bloque en `db/queries.sql`:**
```sql
-- name: ListTareasByRangoFechas :many
-- Para pintar el calendario entre dos fechas
SELECT id_tarea, id_lista, titulo, fecha_inicio, fecha_finalizacion, descripcion, completada
FROM tarea
WHERE fecha_finalizacion >= $1 AND fecha_finalizacion <= $2
ORDER BY fecha_finalizacion ASC;
```
- `$1` y `$2` son marcadores de posición de PostgreSQL que previenen cualquier intento de **Inyección SQL**, ya que los valores viajan separados de la sentencia SQL.

---

### 4.3. `db/coneccion_db.go` (Conexión con PostgreSQL vía pgx)
- **Propósito**: Encapsula la apertura de conexión a PostgreSQL y comprueba que el servidor de base de datos responda activamente antes de continuar.

```go
package db

import (
	"context"
	"github.com/jackc/pgx/v5"
)

func AbridDB(ctx context.Context, connStr string) (*pgx.Conn, error) {
	conn, err := pgx.Connect(ctx, connStr)
	if err != nil {
		return nil, err
	}

	// Ping() valida que la base de datos realmente responda
	if err := conn.Ping(ctx); err != nil {
		return nil, err
	}

	return conn, nil
}
```

---

### 4.4. `db/db.go` (Interfaz DBTX e inicializador generado por sqlc)
- **Propósito**: Archivo autogenerado por `sqlc`. Define la interfaz `DBTX` (que abstrae tanto una conexión directa `*pgx.Conn`, un pool de conexiones o una transacción `pgx.Tx`) y la estructura `Queries`.

```go
type DBTX interface {
	Exec(context.Context, string, ...interface{}) (pgconn.CommandTag, error)
	Query(context.Context, string, ...interface{}) (pgx.Rows, error)
	QueryRow(context.Context, string, ...interface{}) pgx.Row
}

func New(db DBTX) *Queries {
	return &Queries{db: db}
}
```
- Incluye además el método `WithTx(tx pgx.Tx) *Queries` para ejecutar múltiples consultas dentro de una transacción atómica si se requiere.

---

### 4.5. `db/models.go` (Estructuras de dominio generadas por sqlc)
- **Propósito**: Archivo autogenerado por `sqlc` que traduce cada tabla de `db/schema.sql` en un `struct` de Go.

```go
type Tarea struct {
	IDTarea           int32
	IDLista           int32
	Titulo            string
	FechaInicio       pgtype.Timestamp
	FechaFinalizacion pgtype.Timestamp
	Descripcion       pgtype.Text
	Completada        bool
}
```
- **Por qué se usa `pgtype.Timestamp` y `pgtype.Text`**: En SQL, las columnas `fecha_inicio`, `fecha_finalizacion` y `descripcion` permiten valores `NULL`. Como en Go un `string` o `time.Time` no puede ser `nil`, `pgx/v5/pgtype` provee structs con un campo booleano `Valid` (`true` si tiene valor, `false` si es `NULL` en la base de datos) y soporte nativo de serialización JSON.

---

### 4.6. `db/queries.sql.go` (Métodos CRUD en Go generados por sqlc)
- **Propósito**: Contiene la implementación en Go de las 19 consultas definidas en `db/queries.sql`.

**Explicación de un método generado (`CreateTarea`):**
```go
type CreateTareaParams struct {
	IDLista           int32
	Titulo            string
	FechaInicio       pgtype.Timestamp
	FechaFinalizacion pgtype.Timestamp
	Descripcion       pgtype.Text
}

func (q *Queries) CreateTarea(ctx context.Context, arg CreateTareaParams) (Tarea, error) {
	row := q.db.QueryRow(ctx, createTarea,
		arg.IDLista,
		arg.Titulo,
		arg.FechaInicio,
		arg.FechaFinalizacion,
		arg.Descripcion,
	)
	var i Tarea
	err := row.Scan(
		&i.IDTarea,
		&i.IDLista,
		&i.Titulo,
		&i.FechaInicio,
		&i.FechaFinalizacion,
		&i.Descripcion,
		&i.Completada,
	)
	return i, err
}
```
- Recibe un struct de parámetros (`CreateTareaParams`), ejecuta `QueryRow` con la constante SQL y mapea cada columna devuelta por `RETURNING` directamente en la estructura `Tarea` mediante `row.Scan(...)`.

---

### 4.7. `db/db_test.go` (Pruebas de Integración de Base de Datos)
- **Propósito**: Prueba de forma secuencial y automatizada el ciclo de vida CRUD completo directamente contra PostgreSQL (sin pasar por HTTP), verificando que todas las consultas de `sqlc` funcionen según lo esperado.

```go
func TestQueries_CRUD(t *testing.T) {
	// ... abre conexión a BD ...
	var user CreateUserRow
	t.Run("Tabla_Usuario", func(t *testing.T) {
		user = testUsuario(ctx, t, queries)
	})

	var materia ListaTarea
	t.Run("Tabla_ListaTarea_Materia", func(t *testing.T) {
		materia = testMateria(ctx, t, queries, user.IDUsuario)
	})

	var tarea Tarea
	t.Run("Tabla_Tarea", func(t *testing.T) {
		tarea = testTarea(ctx, t, queries, materia.IDLista)
	})

	var horas HorasTrabajada
	t.Run("Tabla_HorasTrabajadas", func(t *testing.T) {
		horas = testHorasTrabajadas(ctx, t, queries, tarea.IDTarea)
	})

	t.Run("Limpieza_Delete_Secuencial", func(t *testing.T) {
		testLimpieza(ctx, t, queries, horas.IDHoras, tarea.IDTarea, materia.IDLista, user.IDUsuario)
	})
}
```
- **Explicación**:
  - Como existe dependencia de claves foráneas (`Usuario` -> `Materia` -> `Tarea` -> `HorasTrabajadas`), el test principal encadena sub-tests (`t.Run`) pasando el ID recién creado de cada entidad padre hacia la prueba de su entidad hija.
  - Al final, `testLimpieza` elimina en orden inverso las horas, la tarea y la materia, verificando con `errors.Is(err, pgx.ErrNoRows)` o longitud de listas igual a `0` que los datos realmente desaparecieron de la base de datos.

---

## 5. Capa de Controladores HTTP (`handlers/`)

Los archivos dentro de `handlers/` reciben las peticiones HTTP (`http.ResponseWriter`, `*http.Request`) junto con el puntero `*db.Queries`, validan los datos de entrada, invocan la capa `db` y devuelven respuestas JSON o HTML.

### 5.1. `handlers/usuario_handler.go` (API de Usuarios y Vistas Templ)
- **Propósito**: Maneja tanto los endpoints REST de usuarios como los controladores de vistas HTML renderizadas con Templ.

#### Funciones que contiene:

1. **`CreateUsuario(w, r, q)` (Híbrido Formulario HTML y JSON REST)**:
   Detecta el encabezado `Content-Type` de la petición para soportar dos clientes distintos en la misma ruta `POST /usuarios`:
   ```go
   // 1. Caso envío desde Formulario HTML nativo (sin JavaScript)
   if strings.Contains(r.Header.Get("Content-Type"), "application/x-www-form-urlencoded") {
       if err := r.ParseForm(); err != nil {
           w.Header().Set("Content-Type", "text/html; charset=utf-8")
           view.CreateUsuarioPage("Error al procesar el formulario de alta").Render(r.Context(), w)
           return
       }
       newUsuario.Nombre = strings.TrimSpace(r.FormValue("nombre"))
       // ... valida campos vacíos e inserta en BD ...
       http.Redirect(w, r, "/usuarios?creado="+strconv.Itoa(int(userCreated.IDUsuario)), http.StatusSeeOther)
       return
   }

   // 2. Caso petición JSON (para API REST y pruebas automatizadas Hurl)
   err := json.NewDecoder(r.Body).Decode(&newUsuario)
   // ... valida, inserta en BD y responde HTTP 201 Created con JSON ...
   ```
   - Si proviene del formulario web (`create_usuario.templ`), extrae los campos con `r.FormValue()`, y si ocurre un error vuelve a renderizar la página del formulario mostrando el mensaje de alerta; si tiene éxito, realiza una redirección HTTP `303 See Other` hacia `/usuarios?creado=ID`.
   - Si proviene de un cliente API (`application/json`), decodifica el cuerpo JSON y devuelve código `201 Created` con el objeto creado en JSON.

2. **`GetUserByEmail(w, r, q)` (`GET /usuarios/email/{email}`)**:
   Extrae el parámetro de ruta con `r.PathValue("email")`, valida que no sea vacío, consulta `q.GetUserByEmail` y responde `200 OK` en JSON o `404 Not Found`.

3. **`GetUserById(w, r, q)` (`GET /usuarios/{id}`)**:
   Extrae `{id}` de la URL, lo convierte a entero de 32 bits (`strconv.ParseInt(idStr, 10, 32)`), consulta `q.GetUserByID` y devuelve los datos públicos del usuario en JSON.

4. **`UsuariosView(w, r, q)` (`GET /` y `GET /usuarios`)**:
   Obtiene todos los usuarios registrados mediante `q.ListUsers(r.Context())`, verifica si en la URL viene el query param `?creado=ID` o `?activo=ID` para resaltar qué usuario tiene la sesión activa en la interfaz y renderiza el componente Templ `view.LayoutPrincipal(...)`.

5. **`CreateUsuarioView(w, r)` (`GET /usuarios/nuevo`)**:
   Renderiza el formulario HTML de alta de usuario (`view.CreateUsuarioPage("")`).

6. **`LoginView(w, r)` (`GET /login`) y `LoginSubmit(w, r, q)` (`POST /login`)**:
   - `LoginView` lee el query param opcional `?email=...` para prellenar el correo del usuario seleccionado y muestra la pantalla de ingreso de contraseña.
   - `LoginSubmit` busca al usuario por email en PostgreSQL y compara la contraseña ingresada (`u.Contrasena != contrasena`). Si falla, vuelve a mostrar el login con el mensaje `"Contraseña incorrecta o usuario no encontrado"`; si coincide, redirige a `/tp6?nombre=NombreUsuario`.

7. **`TP6View(w, r)` (`GET /tp6`, `GET /calendario`, `GET /materias`)**:
   Renderiza el componente `view.TP6Aviso(nombre)` informando que el tablero de materias y calendario se implementarán en la entrega TP6.

---

### 5.2. `handlers/materia_handler.go` (API CRUD de Materias / Listas)
- **Propósito**: Administra el ciclo CRUD de la entidad `ListaTarea` (Materias) a través de respuestas JSON.

#### Funciones que contiene:
- **`CreateMateria` (`POST /materias`)**: Decodifica `db.CreateMateriaParams`, valida que `Titulo` no esté vacío y que `IDUsuario > 0`, inserta en la tabla `lista_tarea` y devuelve `201 Created`.
- **`GetListMateriasByUsuario` (`GET /materias/{id}`)**: Toma el `{id}` del usuario desde la ruta, lo convierte a `int32` y retorna un arreglo JSON con todas las materias pertenecientes a ese usuario (`200 OK`).
- **`PutUpdateMateria` (`PUT /materias/{id}`)**: Decodifica `db.UpdateMateriaParams` del body JSON, valida `IDLista > 0` y `Titulo` no vacío, actualiza el nombre de la materia y retorna el registro actualizado (`200 OK`).
- **`DeleteMateria` (`DELETE /materias/{id}`)**: Parsea el ID de la materia y ejecuta `q.DeleteMateria`, respondiendo con estado `204 No Content`.

---

### 5.3. `handlers/tarea_handler.go` (API CRUD de Tareas y Rango de Fechas)
- **Propósito**: Gestiona las operaciones sobre la tabla `tarea`, incluyendo búsquedas por materia y filtrado por ventana de fechas para el calendario.

#### Funciones que contiene:
- **`CreateTarea` (`POST /tareas`)**: Decodifica `db.CreateTareaParams` (el driver `pgtype` convierte automáticamente las cadenas ISO-8601/RFC3339 del JSON en `pgtype.Timestamp`), valida `Titulo` e `IDLista > 0` y responde `201 Created`.
- **`GetTareaById` (`GET /tareas/{id}`)**: Busca y retorna una tarea puntual por su `id_tarea`.
- **`GetListTareasByMateria` (`GET /tareas/materia/{id}`)**: Devuelve todas las tareas asociadas a una materia (`id_lista`).
- **`GetListTareasByRangoFechas` (`GET /tareas/fechas`)**:
  Lee dos parámetros de consulta (`Query Params`) desde la URL y los convierte a `pgtype.Timestamp`:
  ```go
  fechaInicioStr := r.URL.Query().Get("FechaFinalizacion")
  fechaFinalStr := r.URL.Query().Get("FechaFinalizacion_2")

  fechaInicio, err := time.Parse(time.RFC3339, fechaInicioStr)
  if err != nil {
      http.Error(w, "Fecha final inválida", http.StatusBadRequest)
      return
  }

  fechaFinal, err := time.Parse(time.RFC3339, fechaFinalStr)
  if err != nil {
      http.Error(w, "Fecha final_2 inválida", http.StatusBadRequest)
      return
  }

  params := db.ListTareasByRangoFechasParams{
      FechaFinalizacion:   pgtype.Timestamp{Time: fechaInicio, Valid: true},
      FechaFinalizacion_2: pgtype.Timestamp{Time: fechaFinal, Valid: true},
  }
  ```
  - Utiliza el estándar `time.RFC3339` (ej. `2026-10-19T15:30:00Z`) para parsear las fechas enviadas en la URL y consultar todas las tareas cuya fecha de finalización caiga dentro de ese intervalo.
- **`PutUpdateTarea` (`PUT /tareas/{id}`)**: Actualiza título, fechas, descripción y el booleano `Completada` de una tarea existente.
- **`DeleteTarea` (`DELETE /tareas/{id}`)**: Elimina una tarea por su ID y devuelve `204 No Content`.

---

### 5.4. `handlers/horas_trabajadas_handler.go` (API CRUD de Sesiones de Estudio)
- **Propósito**: Controla el registro y consulta de sesiones de tiempo (`horas_trabajadas`) invertidas en cada tarea.

#### Funciones que contiene:
- **`CreateHorasTrabajadas` (`POST /horas`)**: Recibe `IDTarea`, `TimestampInicio` y `TimestampFin` en JSON, valida que `IDTarea > 0` y guarda la sesión en la base de datos (`201 Created`).
- **`GetListHorasByTarea` (`GET /horas/{id}`)**: Retorna todas las sesiones de tiempo registradas para la tarea `{id}`, ordenadas desde la más reciente a la más antigua.
- **`GetListHorasByRangoFechas` (`GET /horas/fechas`)**: Lee los query params `FechaInicio` y `FechaInicio_2` en formato `RFC3339` y devuelve las sesiones iniciadas dentro de ese período para construir estadísticas de estudio.
- **`PutUpdateHorasTrabajadas` (`PUT /horas/{id}`)**: Permite corregir la hora de inicio o finalización de una sesión específica (`IDHoras`).
- **`DeleteHorasTrabajadas` (`DELETE /horas/{id}`)**: Borra un registro de horas trabajadas y responde `204 No Content`.

---

## 6. Capa de Vistas Server-Side Rendering (`view/`)

El directorio `view/` utiliza **Templ** para definir componentes HTML tipados en archivos `.templ`, los cuales son compilados a código Go nativo en sus respectivos archivos `*_templ.go`.

### 6.1. `view/layout.templ` y `view/layout_templ.go`
- **Propósito**: Define la estructura global del documento HTML y la vista principal de bienvenida/gestión de usuarios.
- **Componentes en `view/layout.templ`**:
  1. **`LayoutBase(titulo string, usuarioActivo string)`**:
     Renderiza el `<head>` (con Google Fonts `Caveat` e `Inter`, íconos de FontAwesome y `/css/style.css`), la barra de navegación superior (`<header class="site-header">`) con el indicador de sesión activa o `"Sin sesión activa"`, las pestañas hacia `/usuarios`, `/calendario` y `/materias`, y el contenedor `<main class="main-wrapper">` donde inyecta `{ children... }`.
  2. **`LayoutPrincipal(usuarios []db.Usuario, usuarioActivoID int32, usuarioActivoNombre string)`**:
     Envuelve el contenido dentro de `@LayoutBase(...)`, muestra el encabezado de `go_tasks`, invoca al componente `@ListaUsuarios(usuarios, usuarioActivoID)` y dibuja el botón inferior `"Crear Nuevo Usuario"` que enlaza a `/usuarios/nuevo`.

---

### 6.2. `view/usuario.templ` y `view/usuario_templ.go`
- **Propósito**: Contiene los componentes visuales relacionados con el listado de tarjetas de usuario, el formulario de verificación de contraseña (Login) y la pantalla de aviso para la entrega TP6.
- **Componentes en `view/usuario.templ`**:
  1. **`ListaUsuarios(usuarios []db.Usuario, usuarioActivoID int32)`**:
     ```templ
     templ ListaUsuarios(usuarios []db.Usuario, usuarioActivoID int32) {
         if len(usuarios) == 0 {
             <div class="empty-state-card">
                 <i class="fa-solid fa-users-slash empty-state-icon"></i>
                 <h3 class="empty-state-title">No se han creado usuarios</h3>
                 ...
             </div>
         } else {
             <section class="users-container">
                 for _, u := range usuarios {
                     <article class={ "user-card", templ.KV("active", u.IDUsuario == usuarioActivoID) }>
                         ...
     ```
     - Si la base de datos no tiene usuarios, renderiza una tarjeta informativa (`.empty-state-card`).
     - Si existen usuarios, itera sobre el slice `[]db.Usuario` creando una tarjeta por cada uno. Usa `templ.KV("active", u.IDUsuario == usuarioActivoID)` para agregar dinámicamente la clase CSS `active` al usuario en sesión, y muestra el botón `"Sesión Activa"` o el enlace `"Iniciar Sesión"` (`/login?email=...`).
  2. **`LoginPage(emailPrellenado string, errorMsg string)`**:
     Muestra la tarjeta de inicio de sesión con el campo de correo prellenado, el campo de contraseña y un bloque condicional `if errorMsg != ""` que despliega una alerta roja si las credenciales fallaron. Envía los datos mediante `POST /login`.
  3. **`TP6Aviso(usuarioNombre string)`**:
     Muestra el cartel para las secciones en construcción (`"Esta funcionalidad se implementará en la próxima entrega TP6"`) y un botón con `onclick="if (history.length > 1) { history.back(); } else { window.location.href='/usuarios'; }"` para regresar a la vista previa.

---

### 6.3. `view/create_usuario.templ` y `view/create_usuario_templ.go`
- **Propósito**: Define la página con el formulario de registro de nuevos usuarios (`CreateUsuarioPage(errorMsg string)`).
- **Explicación**:
  - Renderiza un `<form action="/usuarios" method="POST">` con 4 campos obligatorios (`required`): `nombre`, `apellido`, `email` y `contrasena`.
  - No requiere JavaScript en el navegador: el envío se realiza de forma nativa mediante `application/x-www-form-urlencoded` hacia el handler `CreateUsuario`.

---

## 7. Capa de Recursos Estáticos y Maquetas (`static/`)

### 7.1. `static/css/style.css`
- **Propósito**: Hoja de estilos CSS nativa de toda la aplicación (1379 líneas, sin dependencia de frameworks externos como Bootstrap o Tailwind).
- **Estructura interna**:
  - **Variables CSS (`:root`)**: Define la paleta de colores *Dark Mode* (`--color-bg: #0f1115`, `--color-card: #191b22`), colores de acento ámbar (`--color-primary: #f59e0b`), tipografías (`Inter` y `Caveat`), bordes redondeados y sombras.
  - **Estilos de Navegación (`.site-header`, `.nav-tabs`, `.user-badge`)**: Barra superior fija con efecto *glassmorphism* (`backdrop-filter: blur`) y pestañas interactivas.
  - **Tarjetas de Usuario y Estados Vacíos (`.users-container`, `.user-card`, `.empty-state-card`)**: Grilla adaptable (`CSS Grid`) para las tarjetas de usuarios y resaltado de sesión activa.
  - **Formularios y Alertas (`.form-card`, `.login-card`, `.form-control`, `.alert-error`)**: Diseño estilizado de inputs, botones y mensajes de validación.
  - **Diseño de Tablero Kanban, Calendario y Cartel TP6 (`.tp6-card`, `.tp6-title-big`)**: Estilos preparados tanto para el aviso actual del TP6 como para las vistas completas de columnas y calendario.

### 7.2. Maquetas HTML Estáticas (`index.html`, `usuarios.html`, `calendario.html`, `materias.html`)
- **Propósito**: Prototipos HTML puros ubicados en `static/` que sirvieron como base de diseño visual (Entregable 1) antes de su migración a plantillas dinámicas SSR con Templ, y que continúan disponibles bajo el prefijo `/static/`:
  - `static/index.html`: Maqueta estática de la pantalla de Login.
  - `static/usuarios.html`: Maqueta estática con usuarios de ejemplo y formulario de registro.
  - `static/calendario.html` y `static/materias.html`: Maquetas estáticas con la barra de navegación y el cartel informativo de la próxima entrega TP6.

---

## 8. Automatización, Scripts y Testing End-to-End (`tests/`)

### 8.1. `tests/server.sh`, `tests/start.sh` y `tests/stop.sh`
- **Propósito**: `tests/server.sh` es el administrador de procesos del proyecto. Permite levantar el entorno completo en segundo plano sin bloquear la terminal y gestionarlo mediante subcomandos (`start`, `stop`, `stop-all`, `restart`, `status`, `logs`).
- **`tests/start.sh` y `tests/stop.sh`**: Son scripts envoltorios de 4 líneas que delegan directamente en `server.sh start` y `server.sh stop`.

**Explicación del flujo de arranque en `tests/server.sh` (`start_server`):**
```bash
    # 1. Verificar si ya está corriendo revisando bin/server.pid
    # 2. Iniciar contenedor PostgreSQL en Docker y esperar healthcheck (wait_for_db)
    docker compose up -d db
    wait_for_db

    # 3. Generar código Go desde SQL con sqlc
    sqlc generate

    # 4. Compilar el servidor Go en bin/server
    go build -o "$BIN_FILE" .

    # 5. Levantar servidor en segundo plano con nohup y guardar su PID
    nohup "$BIN_FILE" > "$LOG_FILE" 2>&1 &
    PID=$!
    echo "$PID" > "$PID_FILE"
```
- Después de lanzar el proceso en segundo plano, ejecuta un bucle de verificación con `curl -s http://localhost:8080/` para confirmar que el servidor esté respondiendo peticiones HTTP y no haya fallado al arrancar.

---

### 8.2. `tests/test.sh`
- **Propósito**: Ejecuta de principio a fin todas las pruebas del sistema en un entorno limpio y aislado.

**Flujo paso a paso de `tests/test.sh`:**
```bash
# Registra cleanup automático al salir (incluso si un test falla)
trap cleanup EXIT

# 1. Limpia contenedores y volúmenes previos
docker compose down -v --remove-orphans

# 2. Genera el código de la capa db con sqlc
sqlc generate

# 3. Levanta PostgreSQL limpio y espera a que esté "healthy"
docker compose up -d
esperar_db

# 4. Ejecuta los tests de integración de Go (db/db_test.go) sin caché
go test -v -count=1 ./...

# 5. Reinicia la base de datos desde cero para que los IDs vuelvan a empezar en 1
docker compose down -v
docker compose up -d
esperar_db

# 6. Compila y levanta el servidor HTTP en segundo plano
go build -o bin/test_server_bin .
./bin/test_server_bin &
SERVER_PID=$!

# 7. Espera a que el puerto 8080 responda y ejecuta Hurl
hurl --test tests/request.hurl
```
- **Por qué reinicia la base de datos en el paso 5**: Los tests de Go (`db_test.go`) insertan y borran registros, lo cual incrementa las secuencias `SERIAL` de PostgreSQL. Al recrear el volumen antes de correr Hurl, se garantiza que los primeros registros creados por `request.hurl` tengan siempre `ID = 1`.

---

### 8.3. `tests/request.hurl`
- **Propósito**: Define las pruebas de integración End-to-End de la API HTTP utilizando **Hurl**. Realiza peticiones reales a `http://localhost:8080` y comprueba tanto los códigos de estado HTTP como el contenido exacto del JSON devuelto mediante expresiones `jsonpath`.

**Explicación de un bloque de prueba en `tests/request.hurl`:**
```hurl
# 1. createTarea
POST http://localhost:8080/tareas
Content-Type: application/json

{
    "IDLista": 1,
    "Titulo": "trabajoPracticoMongo",
    "FechaInicio": "2026-10-19T15:30:00Z",
    "FechaFinalizacion": "2026-11-19T15:30:00Z",
    "Descripcion": "Trabajo practico de prueba de bases NoSQL"
}

HTTP 201
[Asserts]
jsonpath "$.IDTarea" == 1 
jsonpath "$.IDLista" == 1 
jsonpath "$.Titulo" == "trabajoPracticoMongo"
jsonpath "$.FechaInicio" == "2026-10-19T15:30:00"
jsonpath "$.FechaFinalizacion" == "2026-11-19T15:30:00"
jsonpath "$.Descripcion" == "Trabajo practico de prueba de bases NoSQL"
```
- **Cobertura completa de `tests/request.hurl`**:
  1. **Usuarios**: Crea un usuario (`POST /usuarios`), lo busca por email (`GET /usuarios/email/carlos@gmail.com`) y lo busca por ID (`GET /usuarios/1`).
  2. **Materias**: Crea una materia para el usuario 1 (`POST /materias`), lista las materias del usuario 1 (`GET /materias/1`) y actualiza su título (`PUT /materias/1`).
  3. **Tareas**: Crea una tarea en la materia 1 (`POST /tareas`), la obtiene por ID (`GET /tareas/1`), lista las tareas de la materia 1 (`GET /tareas/materia/1`), filtra tareas por rango de fechas (`GET /tareas/fechas?...`) y actualiza la tarea marcándola como `Completada: true` (`PUT /tareas/1`).
  4. **Horas Trabajadas**: Registra una sesión de horas en la tarea 1 (`POST /horas`), lista las horas de la tarea 1 (`GET /horas/1`), filtra horas por rango de fechas (`GET /horas/fechas?...`) y actualiza la fecha de fin (`PUT /horas/1`).
  5. **Eliminación en cascada / secuencial**: Ejecuta `DELETE /horas/1`, `DELETE /tareas/1` y `DELETE /materias/1`, verificando que los tres devuelvan `HTTP 204`.

---

## 9. Documentación Complementaria (`docs/`) y Binarios (`bin/`)

- **`docs/documentacion_dominio.md`**: Especificación conceptual del modelo de dominio, sus 4 entidades, el diagrama Entidad-Relación (ERD), las reglas de borrado en cascada y la matriz de endpoints CRUD.
- **`docs/documentacion_vistas_templ.md`**: Explicación detallada sobre el motor de plantillas Templ, el flujo de compilación `.templ` -> `*_templ.go`, la inyección de componentes hijos (`{ children... }`) y la resolución de problemas frecuentes de compilación.
- **`bin/server` y `bin/server.pid`**: Directorio generado al iniciar el servidor donde se aloja el ejecutable compilado de Go y el archivo con el identificador de proceso (PID) activo, junto con `server.log` en la raíz para la inspección de eventos en tiempo real.
