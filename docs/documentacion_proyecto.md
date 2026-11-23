# Documentación del Proyecto y Modelo de Dominio — `go_tasks`

## 1. Visión General y Arquitectura

**go_tasks** es una aplicación web orientada a la productividad y gestión del tiempo académico y profesional. Permite a cada usuario administrar sus materias o proyectos en un tablero por columnas, gestionar tareas con fechas de inicio y entrega, y registrar sesiones de estudio mediante un cronómetro integrado a un calendario semanal.

## 2. Base de Datos y Entidades de Dominio

### 2.1. Usuario (`usuario`)
* **Responsabilidad:** Autenticación, control de acceso y aislamiento multiusuario.
* **Atributos (`db/schema.sql`):**
  * `id_usuario` (`SERIAL`, PK): Identificador numérico único secuencial.
  * `nombre` (`VARCHAR(100) NOT NULL`): Nombre del usuario.
  * `apellido` (`VARCHAR(100) NOT NULL`): Apellido del usuario.
  * `email` (`VARCHAR(255) NOT NULL UNIQUE`): Correo electrónico único usado para iniciar sesión.
  * `contrasena` (`VARCHAR(255) NOT NULL`): Credencial de acceso.

### 2.2. Materia (`lista_tarea`)
* **Responsabilidad:** Agrupar tareas bajo un contexto temático común.
* **Atributos (`db/schema.sql`):**
  * `id_lista` (`SERIAL`, PK): Identificador secuencial de la materia.
  * `id_usuario` (`INT NOT NULL`, FK): Referencia al usuario propietario (`ON DELETE CASCADE`).
  * `titulo` (`VARCHAR(150) NOT NULL`): Nombre descriptivo de la materia.

### 2.3. Tarea (`tarea`)
* **Responsabilidad:** Almacenar estado de avance, descripción y marcas temporales para el tablero y el calendario semanal.
* **Atributos (`db/schema.sql`):**
  * `id_tarea` (`SERIAL`, PK): Identificador único de la tarea.
  * `id_lista` (`INT NOT NULL`, FK): Referencia a la materia contenedora (`ON DELETE CASCADE`).
  * `titulo` (`VARCHAR(150) NOT NULL`): Título de la actividad.
  * `fecha_inicio` (`TIMESTAMP NULL`): Fecha y hora opcional de inicio de la tarea.
  * `fecha_finalizacion` (`TIMESTAMP`): Fecha y hora límite de entrega.
  * `descripcion` (`TEXT`): Detalles o notas opcionales.
  * `completada` (`BOOLEAN NOT NULL DEFAULT FALSE`): Indica si la tarea fue completada.

### 2.4. Horas Trabajadas (`horas_trabajadas`)
* **Responsabilidad:** Llevar la trazabilidad temporal del tiempo trabajado por tarea y mostrar los bloques en la agenda semanal.
* **Atributos (`db/schema.sql`):**
  * `id_horas` (`SERIAL`, PK): Identificador único del registro.
  * `id_tarea` (`INT NOT NULL`, FK): Tarea asociada (`ON DELETE CASCADE`).
  * `timestamp_inicio` (`TIMESTAMP NOT NULL`): Instante de inicio de la sesión del cronómetro.
  * `timestamp_fin` (`TIMESTAMP`): Instante de finalización de la sesión.

### 2.5. Almacén de Sesiones en Redis
Las sesiones activas se guardan en **Redis** (`localhost:6379`):
* **Clave en Redis:** `session:<token>`, donde `<token>` es un identificador aleatorio criptográficamente seguro de 32 bytes codificado en hexadecimal (64 caracteres).
* **Valor:** `id_usuario` (`int32`).
* **Expiración (TTL):** `24 horas` (`DuracionSesion`), renovada automáticamente en cada petición autenticada válida.
* **Cookie en el navegador:** `go_tasks_session`

---

## 3. Diagrama Relacional de Entidades (ERD)

```text
+-------------------+       1:N       +-------------------+
|      USUARIO      |----------------<|    LISTA_TAREA    |
+-------------------+                 +-------------------+
| PK id_usuario     |                 | PK id_lista       |
|    nombre         |                 | FK id_usuario     |
|    apellido       |                 |    titulo         |
|    email (UNIQUE) |                 +-------------------+
|    contrasena     |                           |
+-------------------+                           | 1:N
                                                v
+-------------------+       1:N       +-------------------+
|  HORAS_TRABAJADAS |<----------------|       TAREA       |
+-------------------+                 +-------------------+
| PK id_horas       |                 | PK id_tarea       |
| FK id_tarea       |                 | FK id_lista       |
|    timestamp_ini  |                 |    titulo         |
|    timestamp_fin  |                 |    fecha_inicio   |
+-------------------+                 |    fecha_fin      |
                                      |    descripcion    |
                                      |    completada     |
                                      +-------------------+
```

### Reglas de Integridad Referencial
* **Cascada en Eliminación (`ON DELETE CASCADE`):**
  * Si se elimina un registro en `usuario`, se eliminan todas sus materias (`lista_tarea`), sus `tarea` y sus `horas_trabajadas`.
  * Si se elimina una `lista_tarea` (materia), se eliminan automáticamente todas sus `tarea` y las `horas_trabajadas` asociadas.
  * Si se elimina una `tarea`, se borran automáticamente todos sus registros en `horas_trabajadas`.

---

## 4. Consultas de Base de Datos (`sqlc`)

| Entidad | Operación `sqlc` | Tipo | Propósito |
| :--- | :--- | :--- | :--- |
| **Usuario** | `CreateUser` | `:one` | Inserta un nuevo usuario y retorna `id_usuario`, `nombre`, `apellido` y `email`. |
| | `GetUserByEmail` | `:one` | Obtiene usuario completo (incluyendo `contrasena`) por `email` para validar login. |
| | `GetUserByID` | `:one` | Obtiene `id_usuario`, `nombre`, `apellido` y `email` (sin contraseña) para las vistas activas. |
| **Materia** | `CreateMateria` | `:one` | Crea una nueva materia asociada a `id_usuario`. |
| | `ListMateriasByUsuario` | `:many` | Lista todas las materias de un usuario ordenadas por `id_lista ASC`. |
| | `UpdateMateria` | `:one` | Actualiza el `titulo` de una materia por `id_lista`. |
| | `DeleteMateria` | `:exec` | Elimina una materia por `id_lista` (y en cascada sus tareas y horas). |
| **Tarea** | `CreateTarea` | `:one` | Crea una tarea con `completada = FALSE` por defecto. |
| | `GetTareaByID` | `:one` | Obtiene una tarea puntual por `id_tarea` para validar propiedad o completar. |
| | `ListTareasByMateria` | `:many` | Lista las tareas de una materia ordenadas por `id_tarea ASC`. |
| | `UpdateTarea` | `:one` | Actualiza título, fechas, descripción y estado `completada` de una tarea. |
| | `DeleteTarea` | `:exec` | Elimina una tarea por `id_tarea` (y en cascada sus horas). |
| **Horas** | `CreateHorasTrabajadas` | `:one` | Inserta un bloque `timestamp_inicio` – `timestamp_fin` para una tarea. |
| | `ListHorasByTarea` | `:many` | Lista las sesiones de una tarea ordenadas por `timestamp_inicio DESC`. |
| | `DeleteHorasTrabajadas` | `:exec` | Elimina un registro de horas trabajadas por `id_horas`. |

---

## 5. Rutas HTTP y Handlers

| Ruta HTTP | Handler (`archivo`) | Sesión Redis | Queries `sqlc` | Respuesta |
| :--- | :--- | :--- | :--- | :--- |
| `GET /css/` | `http.FileServer("./static/css")` (`main.go`) | No | — | Sirve `style.css` bajo `/css/style.css`. |
| `GET /{$}` y `GET /login` | `LoginView` (`usuario_handler.go`) | Opcional | — | Si hay sesión activa redirige a `/calendario`; si no, renderiza `LoginContent` / `LoginPage`. |
| `POST /login` | `LoginSubmit` (`usuario_handler.go`) | Crea sesión | `GetUserByEmail` | Crea clave en Redis y cookie; redirige a `/calendario` (`HX-Redirect` o `303`). Error: `LoginContent` / `LoginPage` (`401`). |
| `GET /usuarios/nuevo` | `CreateUsuarioView` (`usuario_handler.go`) | No | — | Renderiza `CreateUsuarioContent` (HTMX) o `CreateUsuarioPage`. |
| `POST /usuarios` | `CreateUsuario` (`usuario_handler.go`) | No | `CreateUser` | Valida campos; si tiene éxito vuelve al login (`LoginContent` + `HX-Push-Url: /login` o `303 /login`). |
| `GET /logout` | `LogoutRedisHandler` (`redis_session_handler.go`) | Elimina sesión | — | Borra la clave en Redis (`Del`), expira cookie (`MaxAge: -1`) y redirige a `/login`. |
| `GET /usuario` | `UsuarioView` (`usuario_handler.go`) | Requerida | `GetUserByID` | Renderiza `UsuarioContent` (HTMX) o `UsuarioPage` con datos de la cuenta activa. |
| `GET /calendario` | `CalendarioView` (`calendario_handler.go`) | Requerida | `ListMateriasByUsuario`, `ListTareasByMateria`, `ListHorasByTarea`, `GetUserByID` | Renderiza `CalendarioContent` (HTMX) o `CalendarioPage`. |
| `GET /materias` | `MateriasView` (`materia_handler.go`) | Requerida | `ListMateriasByUsuario`, `ListTareasByMateria`, `GetUserByID` | Renderiza `MateriasContent` (HTMX) o `MateriasPage`. |
| `POST /materias` | `CreateMateria` (`materia_handler.go`) | Requerida | `CreateMateria` + recarga tablero | Renderiza tablero actualizado (`MateriasContent` / `MateriasPage`). |
| `PUT /materias/{id}` | `PutUpdateMateria` (`materia_handler.go`) | Requerida + Owner | `ListMateriasByUsuario`, `UpdateMateria` + recarga tablero | Renderiza tablero actualizado (`MateriasContent` / `MateriasPage`). |
| `DELETE /materias/{id}` | `DeleteMateria` (`materia_handler.go`) | Requerida + Owner | `ListMateriasByUsuario`, `DeleteMateria` + recarga tablero | Renderiza tablero actualizado (`MateriasContent` / `MateriasPage`). |
| `POST /tareas` | `CreateTarea` (`tarea_handler.go`) | Requerida + Owner | `ListMateriasByUsuario`, `CreateTarea` + recarga tablero | Renderiza tablero actualizado (`MateriasContent` / `MateriasPage`). |
| `POST /tareas/{id}/completar` | `CompletarTarea` (`tarea_handler.go`) | Requerida + Owner | `GetTareaByID`, `ListMateriasByUsuario`, `UpdateTarea` + recarga tablero | Marca `completada = true` y renderiza tablero actualizado. |
| `PUT /tareas/{id}` | `PutUpdateTarea` (`tarea_handler.go`) | Requerida + Owner | `GetTareaByID`, `ListMateriasByUsuario`, `UpdateTarea` + recarga tablero | Actualiza la tarea y renderiza tablero actualizado. |
| `DELETE /tareas/{id}` | `DeleteTarea` (`tarea_handler.go`) | Requerida + Owner | `GetTareaByID`, `ListMateriasByUsuario`, `DeleteTarea` + recarga tablero | Elimina la tarea y renderiza tablero actualizado. |
| `POST /horas_trabajadas` | `CreateHorasTrabajadas` (`horas_trabajadas_handler.go`) | Requerida + Owner | `GetTareaByID`, `ListMateriasByUsuario`, `CreateHorasTrabajadas` + recarga calendario | Guarda intervalo de cronómetro y renderiza calendario actualizado. |
| `DELETE /horas/{id}` | `DeleteHorasTrabajadas` (`horas_trabajadas_handler.go`) | Requerida + Owner | `ListMateriasByUsuario`, `ListTareasByMateria`, `ListHorasByTarea`, `DeleteHorasTrabajadas` + recarga calendario | Elimina el bloque horario y renderiza calendario actualizado. |

---

## 6. Vistas Principales de Templ

* **`LayoutAuth` (`#auth-content`):** usado en `login.templ` y `create_usuario.templ` para usuarios no autenticados. Permite alternar entre Login y Registro. Al autenticarse con éxito, `LoginSubmit` envía `HX-Redirect: /calendario` para hacer la transición al layout base.
* **`LayoutBase` (`#main-content`):** usado en `usuario.templ`, `calendario.templ` y `materias.templ`. Mantiene una barra superior persistente con el usuario activo (`user-badge`), las pestañas `Usuario`, `Calendario`, `Materias` y el enlace `Salir` (`/logout`).
