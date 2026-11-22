package db

// =============================================================================
// AGREGADO PARA REDIS: CONEXIÓN AL SERVIDOR REDIS (db/coneccion_redis.go)
// =============================================================================
// Archivo que contiene el método para inicializar y validar la conexión
// con Redis, siguiendo la misma convención que db/coneccion_db.go.
// =============================================================================

import (
	"context"

	"github.com/redis/go-redis/v9"
)

// AbrirRedis crea un cliente de Redis apuntando a la dirección indicada (ej. "localhost:6379")
// y verifica mediante Ping() que el servidor Redis esté activo y respondiendo.
func AbrirRedis(ctx context.Context, addr string) (*redis.Client, error) {
	// Inicializa el pool de conexiones del cliente Redis con la dirección recibida.
	// Password vacío y DB 0 son los valores por defecto de Redis en desarrollo.
	rdb := redis.NewClient(&redis.Options{
		Addr: addr,
	})

	// Ping() envía el comando PING a Redis para confirmar que la conexión es válida
	// antes de devolver el cliente al servidor principal.
	if err := rdb.Ping(ctx).Err(); err != nil {
		// Si no responde, cerramos el cliente para no dejar sockets abiertos y devolvemos el error.
		_ = rdb.Close()
		return nil, err
	}

	return rdb, nil
}
