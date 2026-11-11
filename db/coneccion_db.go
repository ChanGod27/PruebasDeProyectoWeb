package db

/*Archivo que contiene el metodo para la conexion a la base de datos*/

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
