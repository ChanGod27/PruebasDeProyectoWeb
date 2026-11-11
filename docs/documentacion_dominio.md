# Documentación del Modelo de Dominio — `go_tasks`

## 1. Visión General del Sistema
**go_tasks** es una plataforma orientada a la productividad y gestión del tiempo académico y profesional. Su objetivo central es permitir a los usuarios organizar asignaturas o áreas de trabajo en tableros visuales tipo columnas, gestionar tareas con plazos temporales, y registrar sesiones de estudio o trabajo mediante intervalos de tiempo (cronómetro) vinculados a cada tarea.

---

## 2. Entidades de Dominio

### 2.1. Usuario (`usuario`)
Representa al usuario registrado en la plataforma y dueño de sus propios espacios de trabajo.
* **Responsabilidad:** Autenticación, control de acceso y separación multiusuario. Todas las listas y tareas pertenecen a un usuario específico.
* **Atributos:**
  * `id_usuario` (PK): Identificador numérico único y secuencial.
  * `nombre` / `apellido`: Datos para el perfil.
  * `email`: Correo electrónico único utilizado como identificador para el inicio de sesión.
  * `contrasena`: Credencial de acceso.

### 2.2. Materia / Proyecto (`lista_tarea`)
Representa una columna o tablero (por ejemplo, una materia universitaria, proyecto o lista de pendientes generales).
* **Responsabilidad:** Agrupar tareas bajo un contexto temático común.
* **Atributos:**
  * `id_lista` (PK): Identificador secuencial de la lista/materia.
  * `id_usuario` (FK): Referencia al usuario propietario (`ON DELETE CASCADE`).
  * `titulo`: Nombre descriptivo de la materia o columna (ej. *"Programación Web"*, *"Bases de Datos"*).

### 2.3. Tarea (`tarea`)
Es la unidad principal de acción dentro de una materia. Representa una entrega, estudio, trabajo práctico o actividad puntual.
* **Responsabilidad:** Contener el estado de avance, descripciones y las marcas de tiempo que permiten su visualización en tableros y calendarios.
* **Atributos:**
  * `id_tarea` (PK): Identificador único de la tarea.
  * `id_lista` (FK): Relación obligatoria con la materia a la que pertenece (`ON DELETE CASCADE`).
  * `titulo`: Nombre de la actividad.
  * `fecha_inicio`: Timestamp opcional que indica el momento a partir del cual se puede comenzar la tarea.
  * `fecha_finalizacion`: Timestamp límite para completar la tarea. Es el campo clave para ordenar el calendario y determinar en el frontend si una tarea no completada se encuentra aplazada/vencida.
  * `descripcion`: Detalle en texto sobre requerimientos o anotaciones.
  * `completada`: Booleano (`TRUE`/`FALSE`) que representa si la tarea fue finalizada.

### 2.4. Horas Trabajadas / Sesión de Estudio (`horas_trabajadas`)
Es una sub-entidad que registra los bloques temporales dedicados efectivamente a la ejecución de una tarea (generados por cronómetro o carga manual).
* **Responsabilidad:** Llevar la trazabilidad temporal del tiempo invertido por tarea, permitiendo generar métricas de estudio y visualizar bloques en vistas de agenda.
* **Atributos:**
  * `id_horas` / `id_registro` (PK): Identificador propio de la sesión para permitir su posterior modificación o eliminación puntual.
  * `id_tarea` (FK): Tarea sobre la cual se consumió el tiempo (`ON DELETE CASCADE`).
  * `timestamp_inicio`: Momento exacto en que inició la sesión de trabajo.
  * `timestamp_fin`: Momento en que concluyó la sesión (permite calcular la duración total: `fin - inicio`).

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
|  HORAS_TRABAJADAS |>----------------|       TAREA       |
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
  * Si se elimina un `usuario`, se destruyen todas sus materias, tareas y horas registradas.
  * Si se elimina una `lista_tarea` (materia), se eliminan automáticamente todas sus `tareas` asociadas y las `horas_trabajadas` vinculadas a ellas.
  * Si se elimina una `tarea`, se borran sus respectivas sesiones en `horas_trabajadas`.

---

## 4. Matriz de Operaciones CRUD y Propósito en el Dominio

| Entidad | Operación SQLC | URL (Endpoint) | Propósito en la Aplicación |
| :--- | :--- | :--- | :--- |
| **Usuario** | `CreateUser` | `POST /usuarios` | Registro de una nueva cuenta con validación de email único. |
| | `GetUserByEmail` | `GET /usuarios?email={email}` | Búsqueda para autenticación. |
| | `GetUserByID` | `GET /usuarios/{id}` | Carga de perfil del usuario logueado. |
| **Materia** | `CreateMateria` | `POST /materias` | Creación de una nueva columna en el tablero. |
| | `ListMateriasByUsuario` | `GET /materias/{usuario_id}` | Renderizado de las columnas del tablero del usuario activo. |
| | `UpdateMateria` | `PUT /materias/{id}` | Edición del nombre de la columna. |
| | `DeleteMateria` | `DELETE /materias/{id}` | Eliminación de una materia y todo su contenido. |
| **Tarea** | `CreateTarea` | `POST /tareas` | Creación de tarea rápida desde columna o modal de calendario. |
| | `GetTareaByID` | `GET /tareas/{id}` | Consulta completa de detalles para abrir modal de edición/historial. |
| | `ListTareasByMateria` | `GET /tareas/materia/{materia_id}` | Poblado de tarjetas dentro de una columna específica. |
| | `ListTareasByRangoFechas` | `GET /tareas?FechaFinalizacion={f1}&FechaFinalizacion_2={f2}` | Alimentación de eventos para las vistas semanal/mensual del calendario. |
| | `UpdateTarea` | `PUT /tareas/{id}` | Actualización completa (marcar/desmarcar checkbox, aplazar fecha de fin, editar texto). |
| | `DeleteTarea` | `DELETE /tareas/{id}` | Eliminación de tareas descartadas o completadas. |
| **Horas** | `CreateHorasTrabajadas` | `POST /horas` | Guardado automático al frenar el cronómetro o carga manual de tiempo. |
| | `ListHorasByTarea` | `GET /horas/{tarea_id}` | Historial de sesiones registradas para una tarea específica. |
| | `ListHorasByRangoFechas` | `GET /horas?FechaInicio={f1}&FechaInicio_2={f2}` | Cálculo de horas invertidas para paneles de estadística y bloques de tiempo. |
| | `UpdateHorasTrabajadas` | `PUT /horas/{id}` | Corrección manual de horarios en caso de olvido al apagar el cronómetro. |
| | `DeleteHorasTrabajadas` | `DELETE /horas/{id}` | Eliminación de sesiones cargadas por error. |



---

## 5. Lógica de Negocio y Reglas de Estado
* Estado "Aplazada / Vencida": el modelo de persistencia delega la determinación del retraso a la capa de cliente.
* Una tarea se considera aplazada/vencida si: completada es igual a FALSE (o Falso), y fecha_finalizacion es anterior a la fecha y hora actual (CURRENT_TIMESTAMP).
