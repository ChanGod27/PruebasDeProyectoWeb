#!/bin/bash
set -e

# Asegura que el script siempre se ejecute parado en la raíz del proyecto
PROJECT_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$PROJECT_ROOT"

# Función de limpieza
cleanup() {
    echo "Limpiando servidor y contenedores..."
    if [ -n "$SERVER_PID" ]; then
        kill "$SERVER_PID" 2>/dev/null || true
    fi
    rm -f bin/test_server_bin
    docker compose down -v
}

esperar_db() {
    echo "Esperando a que la base de datos esté lista (Healthcheck)..."
    until [ "$(docker inspect -f '{{.State.Health.Status}}' $(docker compose ps -q db) 2>/dev/null)" = "healthy" ]; do
        sleep 1
    done
}

# Registramos la función de limpieza ante cualquier salida o error
trap cleanup EXIT

echo "1. Limpiando contenedores y volúmenes previos..."
docker compose down -v --remove-orphans

echo "2. Generando código con sqlc..."
sqlc generate

echo "3. Levantando base de datos para tests unitarios..."
docker compose up -d
esperar_db

echo "4. Ejecutando tests unitarios/integración de Go (sin cache)..."
go test -v -count=1 ./...

echo "5. Reiniciando base de datos limpia para tests de API..."
docker compose down -v
docker compose up -d
esperar_db

echo "6. Compilando y levantando el servidor HTTP en segundo plano..."
mkdir -p bin
go build -o bin/test_server_bin .
./bin/test_server_bin &
SERVER_PID=$!

echo "7. Esperando a que el servidor HTTP responda en :8080..."
until curl -s http://localhost:8080/ > /dev/null 2>&1 \
    || curl -s http://localhost:8080/usuarios > /dev/null 2>&1; do
    sleep 1
done

echo "8. Ejecutando tests de API con Hurl..."
hurl --test tests/request.hurl

echo "Pruebas finalizadas con éxito."