package db

/*Archivo que contiene el metodo para la conexion a la base de datos de redis*/

import (
	"context"

	"github.com/redis/go-redis/v9"
)

func AbrirRedis(ctx context.Context, addr string) (*redis.Client, error) {
	rdb := redis.NewClient(&redis.Options{
		Addr: addr,
	})
	if err := rdb.Ping(ctx).Err(); err != nil {
		_ = rdb.Close()
		return nil, err
	}
	return rdb, nil
}
