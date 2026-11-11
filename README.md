## Documentación del Proyecto
- **Persistencia y Dominio**: Especificación del modelo relacional, entidades, matriz CRUD y reglas de negocio en [Documentación de Persistencia](docs/documentacion_dominio.md).
- **Vistas y Templ**: Arquitectura, funcionamiento del ciclo `.templ` $\rightarrow$ `_templ.go` y guía de desarrollo en [Documentación de Vistas con Templ](docs/documentacion_vistas_templ.md).

## Ejecución del Entorno (Base de Datos)
Para levantar la base de datos PostgreSQL con Docker:

Primero hay que dar permiso de ejecucion al archivo test.sh ---> chmod +x test.sh

Luego hay dos formas de ejecutar:  

- "make test"
- "./test.sh" 
