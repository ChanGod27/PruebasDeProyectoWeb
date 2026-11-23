module ServidorEntregable

go 1.26.6

require (

	// Templ
	github.com/a-h/templ v0.3.1118
	//PostgreSQL
	github.com/jackc/pgx/v5 v5.11.0

	// Redis
	github.com/redis/go-redis/v9 v9.23.0
)

require (

	// Requeridas por Redis (go-redis/v9) y Templ
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	// Requeridas por PostgreSQL (pgx/v5)
	github.com/jackc/pgpassfile v1.0.0 // indirect
	github.com/jackc/pgservicefile v0.0.0-20240606120523-5a60cdf6a761 // indirect
	go.uber.org/atomic v1.12.0 // indirect
	golang.org/x/sys v0.50.0 // indirect
	golang.org/x/text v0.29.0 // indirect
)
