# Checklist y Línea de Tiempo de Implementación (`go_tasks` + HTMX + Redis)

Este documento resume el estado actual del proyecto y ordena, paso a paso, todo lo que falta modificar, agregar y conectar hasta dejar la aplicación compilando y funcionando de punta a punta.

---

## Estado actual del proyecto (ya completado)

- [x] **Vistas `.templ` actualizadas para HTMX (Opción A — `HX-Redirect`)** en `views/`:
  - `views/layout.templ`: `LayoutAuth` (sin navbar, contenedor `#auth-content`) y `LayoutBase` (navbar persistente y `#main-content`).
  - `views/login.templ`: `LoginContent` y `LoginPage`.
  - `views/create_usuario.templ`: `CreateUsuarioContent` y `CreateUsuarioPage`.
  - `views/usuario.templ`: `UsuarioContent` y `UsuarioPage` (solo usuario activo).
  - `views/materias.templ`: `MateriasContent` y `MateriasPage` (incluye `MateriaConTareas`, modales `#materia-modal` y `#task-modal`, y checkbox `POST /tareas/{id}/completar`).
  - `views/calendario.templ`: `CalendarioContent` y `CalendarioPage` (incluye `DiaCalendario`, `EventoCalendario` y cronómetro hacia `POST /horas_trabajadas`).
- [x] **Archivos `*_templ.go` viejos eliminados** de `views/` (pendiente regenerar cuando toque compilar).
- [x] **Infraestructura y helpers de sesión en Redis agregados**:
  - `docker-compose.yml`: servicio `redis` (`redis:7-alpine`, puerto `6379`).
  - `go.mod` / `go.sum`: dependencia `github.com/redis/go-redis/v9`.
  - `db/coneccion_redis.go`: `AbrirRedis(ctx, addr)`.
  - `handlers/redis_session_handler.go`: `CrearSesionRedis`, `ObtenerUsuarioIDRedis`, `CerrarSesionRedis`, `RedirigirSinSesion`, `LogoutRedisHandler` y `SessionStatusRedisHandler`.
  - `main.go`: inicialización de `rdb` y rutas `GET /logout` y `GET /sesion/estado`.
- [x] **Limpieza parcial ya realizada**:
  - Eliminada `ListUsers` de `db/queries.sql` (todavía presente en `db/queries.sql.go` hasta correr `sqlc generate`).
  - Eliminadas `UsuariosView` y `TP6View` de `handlers/usuario_handler.go`.
  - Eliminadas las rutas viejas `GET /{$}`, `GET /usuarios`, `GET /tp6`, `GET /calendario` (TP6) y `GET /materias` (TP6) de `main.go`.

---

## Línea de tiempo (orden recomendado de implementación)

### Fase 1 — Base de datos y generación de código (`sqlc` + `templ`)

> **Por qué primero:** mientras `views/*_templ.go` no exista, Go no puede compilar `handlers/`, y mientras no corras `sqlc generate`, `db/queries.sql.go` sigue teniendo la versión vieja antes de tu borrado de `ListUsers`.

- [x] **1.1. Evaluar y agregar nuevas queries filtradas por usuario en `db/queries.sql`**
  - Hoy `ListTareasByRangoFechas` y `ListHorasByRangoFechas` consultan todas las filas sin filtrar por `id_usuario` (si hay más de un usuario en la base, un usuario vería tareas u horas de otro en el calendario).
  - Agregar en `db/queries.sql` (haciendo `JOIN lista_tarea` para filtrar por `id_usuario`):
    - `ListTareasByUsuarioYRangoFechas :many` (para pintar las tareas de la semana del usuario logueado en `/calendario`).
    - `ListHorasByUsuarioYRangoFechas :many` (haciendo `JOIN tarea` + `JOIN lista_tarea` para las horas trabajadas de la semana del usuario).
    - `ListTareasByUsuario :many` (para llenar el `<select>` de tareas pendientes del cronómetro sin hacer N+1 consultas).
    - `MarcarTareaCompletada :one` (o reutilizar `UpdateTarea`, pasando `completada = TRUE` y verificando propiedad del usuario).
- [x] **1.2. Regenerar código SQL con `sqlc`**
  - Ejecutar `sqlc generate` (o `make generate`).
  - Verificar que `db/queries.sql.go` ya no contenga `ListUsers` y sí incluya las queries nuevas.
- [x] **1.3. Generar los archivos Go de las vistas con `templ`**
  - Ejecutar `templ generate`.
  - Verificar que se hayan creado los `*_templ.go` dentro de `views/` y que `go build ./...` (o `go vet ./views/...`) reconozca todos los nuevos componentes (`LoginPage`, `LoginContent`, `MateriasPage`, `CalendarioPage`, `UsuarioPage`, etc.).

---

### Fase 2 — Autenticación y sesiones con Redis (`handlers/usuario_handler.go`)

> **Por qué segundo:** es la puerta de entrada; si el login no crea la cookie en Redis ni redirige a `/calendario`, ninguna vista privada puede probarse.

- [x] **2.1. Actualizar `LoginView` (`GET /` y `GET /login`)**
  - Cambiar su firma para recibir `rdb *redis.Client`: `LoginView(w http.ResponseWriter, r *http.Request, rdb *redis.Client)`.
  - Al inicio, llamar a `ObtenerUsuarioIDRedis(r.Context(), r, rdb)`: si no hay error (ya tiene sesión activa), redirigir directo a `/calendario` (`HX-Redirect` si es HTMX, o `http.Redirect` 303 si es navegación normal).
  - Leer `email := r.URL.Query().Get("email")`.
  - Chequear `r.Header.Get("HX-Request") == "true"`:
    - Si es `true` → renderizar `views.LoginContent(email, "", "")`.
    - Si es `false` → renderizar `views.LoginPage(email, "", "")`.
- [x] **2.2. Actualizar `CreateUsuarioView` (`GET /usuarios/nuevo`)**
  - Chequear `r.Header.Get("HX-Request") == "true"`:
    - Si es `true` → renderizar `views.CreateUsuarioContent("")`.
    - Si es `false` → renderizar `views.CreateUsuarioPage("")`.
- [x] **2.3. Actualizar `CreateUsuario` (`POST /usuarios`)**
  - En la rama de formulario (`application/x-www-form-urlencoded`):
    - Si faltan campos o hay error de BD (ej. email duplicado):
      - Si `r.Header.Get("HX-Request") == "true"` → renderizar `views.CreateUsuarioContent(msgError)` (reemplaza `#auth-content`).
      - Si no → renderizar `views.CreateUsuarioPage(msgError)`.
    - Si el usuario se creó correctamente:
      - Si `r.Header.Get("HX-Request") == "true"` → agregar header `w.Header().Set("HX-Push-Url", "/login")` y renderizar `views.LoginContent("", "", "Usuario creado correctamente. Inicia sesión.")` para volver al login sin iniciar sesión automáticamente.
      - Si no → `http.Redirect(w, r, "/login", http.StatusSeeOther)`.
  - Mantener intacta la rama JSON para que sigan funcionando las pruebas API / `tests/request.hurl`.
- [x] **2.4. Actualizar `LoginSubmit` (`POST /login`)**
  - Cambiar su firma para recibir `rdb *redis.Client`: `LoginSubmit(w http.ResponseWriter, r *http.Request, q *db.Queries, rdb *redis.Client)`.
  - Si el email no existe o `u.Contrasena != contrasena`:
    - Si `r.Header.Get("HX-Request") == "true"` → renderizar `views.LoginContent(email, "Contraseña incorrecta o usuario no encontrado", "")`.
    - Si no → renderizar `views.LoginPage(email, "Contraseña incorrecta o usuario no encontrado", "")`.
  - Si las credenciales son válidas:
    - Llamar a `CrearSesionRedis(r.Context(), w, rdb, u.IDUsuario)`.
    - Si falla Redis, devolver error HTML en `LoginContent`/`LoginPage` (500).
    - Si sale bien:
      - Si `r.Header.Get("HX-Request") == "true"` → `w.Header().Set("HX-Redirect", "/calendario")` y `w.WriteHeader(http.StatusOK)`.
      - Si no → `http.Redirect(w, r, "/calendario", http.StatusSeeOther)`.

---

### Fase 3 — Vista de Usuario Activo (`handlers/usuario_handler.go`)

- [x] **3.1. Crear `UsuarioView(w http.ResponseWriter, r *http.Request, q *db.Queries, rdb *redis.Client)` para `GET /usuario`**
  - Llamar a `idUsuario, err := ObtenerUsuarioIDRedis(r.Context(), r, rdb)`.
  - Si `err != nil` → llamar a `RedirigirSinSesion(w, r)` y `return`.
  - Consultar `uRow, err := q.GetUserByID(r.Context(), idUsuario)`.
  - Mapear `db.GetUserByIDRow` a `db.Usuario` (`IDUsuario`, `Nombre`, `Apellido`, `Email`; la contraseña queda vacía porque no se muestra en la vista).
  - Chequear `r.Header.Get("HX-Request") == "true"`:
    - Si es `true` → `views.UsuarioContent(usuario)`.
    - Si es `false` → `views.UsuarioPage(usuario)`.

---

### Fase 4 — Materias y Tareas (`handlers/materia_handler.go` y `handlers/tarea_handler.go`)

- [x] **4.1. Crear helper interno para armar el tablero (en `handlers/materia_handler.go`)**
  - Escribir una función auxiliar `cargarMateriasConTareas(ctx, q, idUsuario) ([]views.MateriaConTareas, error)` que:
    1. Llame a `q.ListMateriasByUsuario(ctx, idUsuario)`.
    2. Por cada materia (`db.ListaTarea`), llame a `q.ListTareasByMateria(ctx, m.IDLista)`.
    3. Devuelva el slice `[]views.MateriaConTareas`.
- [x] **4.2. Crear `MateriasView(w http.ResponseWriter, r *http.Request, q *db.Queries, rdb *redis.Client)` para `GET /materias`**
  - Validar sesión con `ObtenerUsuarioIDRedis` + `RedirigirSinSesion`.
  - Llamar a `cargarMateriasConTareas(r.Context(), q, idUsuario)`.
  - Chequear `r.Header.Get("HX-Request") == "true"`:
    - Si es `true` → `views.MateriasContent(materiasConTareas)`.
    - Si es `false` → consultar `q.GetUserByID` para obtener el nombre del usuario para la barra superior y renderizar `views.MateriasPage(usuario, materiasConTareas)`.
- [ ] **4.3. Adaptar `CreateMateria` (`POST /materias`) para soportar formularios HTMX**
  - Detectar `strings.Contains(r.Header.Get("Content-Type"), "application/x-www-form-urlencoded")` (manteniendo la rama JSON existente).
  - En la rama formulario/HTMX:
    - Validar sesión con `ObtenerUsuarioIDRedis` + `RedirigirSinSesion` (sacar `IDUsuario` siempre de Redis, nunca de un input).
    - Leer `titulo := strings.TrimSpace(r.FormValue("titulo"))`, validar que no esté vacío y llamar a `q.CreateMateria(...)`.
    - Recargar `cargarMateriasConTareas` y renderizar `views.MateriasContent(materiasConTareas)` (o `MateriasPage` si no vino `HX-Request`).
- [ ] **4.4. Adaptar `CreateTarea` (`POST /tareas`) para soportar formularios HTMX**
  - Detectar `application/x-www-form-urlencoded` (manteniendo la rama JSON existente).
  - En la rama formulario/HTMX:
    - Validar sesión con `ObtenerUsuarioIDRedis` + `RedirigirSinSesion`.
    - Leer `id_lista`, `titulo`, `descripcion`, `fecha_inicio`, `fecha_finalizacion`.
    - **Seguridad (ownership):** verificar que la materia `id_lista` exista y pertenezca a `idUsuario` antes de insertar la tarea.
    - Parsear las fechas de `<input type="datetime-local">` usando el layout `"2006-01-02T15:04"` (si no vienen vacías) a `pgtype.Timestamp`.
    - Llamar a `q.CreateTarea(...)`, recargar `cargarMateriasConTareas` y renderizar `views.MateriasContent(materiasConTareas)`.
- [ ] **4.5. Crear `CompletarTarea(w http.ResponseWriter, r *http.Request, q *db.Queries, rdb *redis.Client)` para `POST /tareas/{id}/completar`**
  - Validar sesión con `ObtenerUsuarioIDRedis` + `RedirigirSinSesion`.
  - Parsear `{id}` con `r.PathValue("id")` y consultar la tarea con `q.GetTareaByID`.
  - **Seguridad (ownership):** verificar que la materia de esa tarea pertenezca a `idUsuario`.
  - Actualizar `completada = true` (usando `MarcarTareaCompletada` o `q.UpdateTarea` conservando título, fechas y descripción).
  - Recargar `cargarMateriasConTareas` y renderizar `views.MateriasContent(materiasConTareas)`.

---

### Fase 5 — Calendario y Cronómetro (`handlers/horas_trabajadas_handler.go` / nuevo `handlers/calendario_handler.go`)

- [ ] **5.1. Crear `CalendarioView(w http.ResponseWriter, r *http.Request, q *db.Queries, rdb *redis.Client)` para `GET /calendario`**
  - Validar sesión con `ObtenerUsuarioIDRedis` + `RedirigirSinSesion`.
  - Obtener la lista de tareas del usuario (las no completadas alimentan el `<select>` del cronómetro en `calendario.templ`).
  - Calcular el rango de la semana actual (Lunes a Sábado, acorde a las 6 columnas de días de `calendario.templ`).
  - Construir `[]views.DiaCalendario` (nombre del día, fecha, resumen, color y `EsHoy: true` para el día actual) y `[]views.EventoCalendario` (`DiaIndex` `0..5` y `Hora` `8..21`) a partir de las tareas y/o horas trabajadas del usuario en esa semana.
  - Chequear `r.Header.Get("HX-Request") == "true"`:
    - Si es `true` → `views.CalendarioContent(tareas, dias, eventos)`.
    - Si es `false` → consultar `q.GetUserByID` y renderizar `views.CalendarioPage(usuario, tareas, dias, eventos)`.
- [ ] **5.2. Alinear la ruta del cronómetro y adaptar `CreateHorasTrabajadas`**
  - Nota importante: `views/calendario.templ` envía el formulario del cronómetro a `POST /horas_trabajadas`, mientras que `main.go` hoy registra `POST /horas`. Elegir una sola ruta (lo más simple sin tocar `views/` es registrar `POST /horas_trabajadas` en `main.go` apuntando al mismo handler, manteniendo `POST /horas` si las pruebas Hurl la usan).
  - En `CreateHorasTrabajadas`, agregar rama `application/x-www-form-urlencoded` (manteniendo la rama JSON):
    - Validar sesión con `ObtenerUsuarioIDRedis` + `RedirigirSinSesion`.
    - Leer `id_tarea`, `timestamp_inicio` y `timestamp_fin` (el script JS de `calendario.templ` los envía en formato ISO `time.RFC3339`).
    - Verificar que la tarea pertenezca a `idUsuario`.
    - Insertar con `q.CreateHorasTrabajadas(...)` y devolver `views.CalendarioContent(...)` actualizado (o recargar los datos del calendario).

---

### Fase 6 — Conectar todas las rutas en `main.go`

- [ ] **6.1. Registrar rutas de vistas y autenticación**
  - `GET /{$}` → `handlers.LoginView(w, r, rdb)` (entrada principal de la app).
  - `GET /login` → `handlers.LoginView(w, r, rdb)`.
  - `POST /login` → `handlers.LoginSubmit(w, r, queries, rdb)`.
  - `GET /usuarios/nuevo` → `handlers.CreateUsuarioView(w, r)` (ya existe, verificar firma).
  - `GET /usuario` → `handlers.UsuarioView(w, r, queries, rdb)`.
  - `GET /calendario` → `handlers.CalendarioView(w, r, queries, rdb)`.
  - `GET /materias` → `handlers.MateriasView(w, r, queries, rdb)`.
- [ ] **6.2. Registrar rutas de acciones HTMX y pasar `rdb` donde corresponda**
  - Actualizar `POST /materias` para pasar `rdb` a `handlers.CreateMateria(w, r, queries, rdb)`.
  - Actualizar `POST /tareas` para pasar `rdb` a `handlers.CreateTarea(w, r, queries, rdb)`.
  - Agregar `POST /tareas/{id}/completar` → `handlers.CompletarTarea(w, r, queries, rdb)`.
  - Agregar `POST /horas_trabajadas` → `handlers.CreateHorasTrabajadas(w, r, queries, rdb)` (y actualizar `POST /horas` con la misma firma).
  - Verificar que `GET /logout` y `GET /sesion/estado` (ya agregadas) sigan registradas.

---

### Fase 7 — Levantar servicios, compilar y probar de punta a punta

- [ ] **7.1. Levantar PostgreSQL y Redis**
  - Ejecutar `docker compose up -d` y comprobar que ambos contenedores pasen su `healthcheck` (`docker compose ps`).
- [ ] **7.2. Compilar el proyecto**
  - Ejecutar `go vet ./...` y `go build -o bin/server .` (o `go run .`).
  - Confirmar que no queden errores de tipos, imports faltantes ni firmas incompatibles.
- [ ] **7.3. Pruebas manuales del flujo completo en el navegador**
  - Entrar a `http://localhost:8080/`: debe mostrar Login sin barra de navegación del dashboard.
  - Navegar a **Crear Nuevo Usuario** (`/usuarios/nuevo`), registrar un usuario y confirmar que vuelve a `/login` con el mensaje de éxito (sin iniciar sesión todavía).
  - Iniciar sesión con credenciales inválidas: debe mostrar error dentro del card sin recargar la página.
  - Iniciar sesión con credenciales válidas: debe crear la cookie `go_tasks_session` en Redis y redirigir con `HX-Redirect` a `/calendario`.
  - Verificar `GET /sesion/estado`: debe devolver `{"autenticado":true,"id_usuario":...}`.
  - Navegar entre **Usuario**, **Calendario** y **Materias**: verificar que cambia `#main-content` sin recargar el `<header>`, que cambia la URL y que al apretar **F5** en cualquier pestaña la página sigue cargando completa con su CSS y navbar.
  - En **Materias**: crear materia, crear tarea y marcarla como completada (comprobar que pasa al acordeón de completadas).
  - En **Calendario**: iniciar el cronómetro, pausar, guardar y verificar que se inserte en PostgreSQL.
  - Hacer click en **Salir** (`/logout`): debe borrar la clave en Redis, limpiar la cookie, volver a `/login` y rechazar el acceso a `/materias`, `/calendario` y `/usuario`.
- [ ] **7.4. (Opcional) Actualizar `Makefile` y `tests/request.hurl`**
  - Agregar `templ generate` dentro de la regla `generate:` del `Makefile` para que `make start` genere tanto `sqlc` como `templ`.
  - Revisar `tests/request.hurl` si se mantienen las pruebas automáticas de la API JSON (y tener en cuenta si las rutas de creación pasan a exigir sesión en Redis).
