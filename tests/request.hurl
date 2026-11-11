# requests.hurl 
# TEST DE CADA ENDPOINT


#TABLA USUARIOS

# 1. createUsuario
POST http://localhost:8080/usuarios
Content-Type: application/json

{
    "Nombre": "Carlos",
    "Apellido": "Galardi",
    "Email": "carlos@gmail.com",
    "Contrasena": "fisica1"
}

HTTP 201
[Asserts]
jsonpath "$.IDUsuario" exists
jsonpath "$.Nombre" == "Carlos"
jsonpath "$.Apellido" == "Galardi"
jsonpath "$.Email" == "carlos@gmail.com"


# 2. getUserByEmail con el email carlos@gmail.com
GET http://localhost:8080/usuarios/email/carlos@gmail.com
HTTP 200
[Asserts]
jsonpath "$.IDUsuario" exists
jsonpath "$.Nombre" == "Carlos"
jsonpath "$.Apellido" == "Galardi"
jsonpath "$.Email" == "carlos@gmail.com"
jsonpath "$.Contrasena" == "fisica1"

# 3. getUserById con el id 1
GET http://localhost:8080/usuarios/1
HTTP 200
[Asserts]
jsonpath "$.IDUsuario" exists
jsonpath "$.Nombre" == "Carlos"
jsonpath "$.Apellido" == "Galardi"
jsonpath "$.Email" == "carlos@gmail.com"


#TABLA ListaTarea (MATERIAS)

# 1. createMateria 
POST http://localhost:8080/materias
Content-Type: application/json

{
    "IDUsuario": 1,
    "Titulo": "BaseDato2"
}

HTTP 201
[Asserts]
jsonpath "$.IDLista" exists
jsonpath "$.IDUsuario" == 1
jsonpath "$.Titulo" == "BaseDato2"


# 2. getListMateriasByUsuario obtenemos la materia del usuario 1
GET http://localhost:8080/materias/1
HTTP 200
[Asserts]
jsonpath "$" count >= 1
jsonpath "$[0].Titulo" == "BaseDato2"


# 3. putUpdateMateria
PUT http://localhost:8080/materias/1
Content-Type: application/json

{
    "IDLista": 1,
    "Titulo": "BaseDato2NombreCambiado"
}

HTTP 200
[Asserts]
jsonpath "$.IDLista" == 1
jsonpath "$.IDUsuario" == 1
jsonpath "$.Titulo" == "BaseDato2NombreCambiado"

#TABLA tarea 

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

# 2. getTareaById obtenemos la tarea con el id 1
GET http://localhost:8080/tareas/1 
HTTP 200
[Asserts]
jsonpath "$.IDTarea" == 1 
jsonpath "$.IDLista" == 1 
jsonpath "$.Titulo" == "trabajoPracticoMongo"
jsonpath "$.FechaInicio" == "2026-10-19T15:30:00"
jsonpath "$.FechaFinalizacion" == "2026-11-19T15:30:00"
jsonpath "$.Descripcion" == "Trabajo practico de prueba de bases NoSQL"
jsonpath "$.Completada" == false 


# 3. getListTareasByMateria obtenemos las tareas de la materia 1 
GET http://localhost:8080/tareas/materia/1 
HTTP 200
[Asserts]
jsonpath "$" count >= 1
jsonpath "$[0].IDTarea" == 1

# 4. getListTareasByRangoFechas
GET http://localhost:8080/tareas/fechas?FechaFinalizacion=2026-10-19T15:30:00Z&FechaFinalizacion_2=2026-12-19T15:30:00Z
HTTP 200
[Asserts]
jsonpath "$" count >= 1
jsonpath "$[0].IDTarea" == 1


# 5. putUpdateTarea con el idTarea 1
PUT http://localhost:8080/tareas/1
Content-Type: application/json

{
    "IDTarea": 1,
    "Titulo": "trabajoPracticoMongoSQL",
    "FechaInicio": "2026-10-19T15:30:00Z",
    "FechaFinalizacion": "2026-11-19T15:30:00Z",
    "Descripcion": "Trabajo practico de prueba de bases NoSQL",
    "Completada": true
}

HTTP 200
[Asserts]
jsonpath "$.IDTarea" == 1 
jsonpath "$.IDLista" == 1 
jsonpath "$.Titulo" == "trabajoPracticoMongoSQL"
jsonpath "$.FechaInicio" == "2026-10-19T15:30:00"
jsonpath "$.FechaFinalizacion" == "2026-11-19T15:30:00"
jsonpath "$.Descripcion" == "Trabajo practico de prueba de bases NoSQL"
jsonpath "$.Completada" == true 


#TABLA horasTrabajadas 

# 1. createHorasTrabajadas
POST http://localhost:8080/horas
Content-Type: application/json

{
    "IDTarea": 1,
    "TimestampInicio": "2026-10-19T15:30:00Z",
    "TimestampFin":  "2026-11-19T15:30:00Z"
}

HTTP 201
[Asserts]
jsonpath "$.IDHoras" == 1
jsonpath "$.IDTarea" == 1
jsonpath "$.TimestampInicio" == "2026-10-19T15:30:00"
jsonpath "$.TimestampFin" == "2026-11-19T15:30:00"

# 2. getListHorasByTarea con idTarea 1 
GET http://localhost:8080/horas/1 
HTTP 200
[Asserts]
jsonpath "$" count >= 1
jsonpath "$[0].IDHoras" == 1


# 3. getListHorasByRangoFechas
GET http://localhost:8080/horas/fechas?FechaInicio=2026-10-19T15:30:00Z&FechaInicio_2=2026-12-19T15:30:00Z
HTTP 200
[Asserts]
jsonpath "$" count >= 1
jsonpath "$[0].IDHoras" == 1


# 4. putUpdateHorasTrabajadas con el IDHoras 1
PUT http://localhost:8080/horas/1
Content-Type: application/json

{
    "IDHoras": 1,
    "TimestampInicio": "2026-10-19T15:30:00Z",
    "TimestampFin":  "2026-12-19T15:30:00Z"
}

HTTP 200
[Asserts]
jsonpath "$.IDHoras" == 1
jsonpath "$.IDTarea" == 1
jsonpath "$.TimestampInicio" == "2026-10-19T15:30:00"
jsonpath "$.TimestampFin" == "2026-12-19T15:30:00"


#Todos los delete 

# 1. deleteHorasTrabajadas
DELETE http://localhost:8080/horas/1
HTTP 204

# 2. deleteTarea
DELETE http://localhost:8080/tareas/1
HTTP 204

#3. deleteMateria
DELETE http://localhost:8080/materias/1
HTTP 204



