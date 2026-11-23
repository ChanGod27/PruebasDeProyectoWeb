#!/bin/bash
set -e

# Asegura que el script siempre se ejecute parado en la raíz del proyecto
PROJECT_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$PROJECT_ROOT"

PID_FILE="bin/server.pid"
BIN_FILE="bin/server"
PORT="8080"

# Función para esperar que Postgres esté listo
wait_for_db() {
    echo "Verificando estado de la base de datos (PostgreSQL)..."
    local attempts=0
    local max_attempts=30

    while [ $attempts -lt $max_attempts ]; do
        if docker compose exec -T db pg_isready -U trabajo_web -d go_tasks_db >/dev/null 2>&1 \
           || [ "$(docker inspect -f '{{.State.Health.Status}}' "$(docker compose ps -q db 2>/dev/null)" 2>/dev/null)" = "healthy" ]; then
            echo "✓ Base de datos lista para aceptar conexiones."
            return 0
        fi
        sleep 1
        attempts=$((attempts + 1))
    done

    echo "⚠ Advertencia: Se agotó el tiempo esperando el healthcheck de la base de datos."
    echo "Intentando continuar con el arranque del servidor..."
    return 0
}

# Función para esperar que Redis esté listo
wait_for_redis() {
    echo "Verificando estado de Redis..."
    local attempts=0
    local max_attempts=30

    while [ $attempts -lt $max_attempts ]; do
        if docker compose exec -T redis redis-cli ping >/dev/null 2>&1 \
           || [ "$(docker inspect -f '{{.State.Health.Status}}' "$(docker compose ps -q redis 2>/dev/null)" 2>/dev/null)" = "healthy" ]; then
            echo "✓ Redis listo para aceptar conexiones."
            return 0
        fi
        sleep 1
        attempts=$((attempts + 1))
    done

    echo "⚠ Advertencia: Se agotó el tiempo esperando el healthcheck de Redis."
    echo "Intentando continuar con el arranque del servidor..."
    return 0
}

start_server() {
    mkdir -p bin

    # Verificar si ya está corriendo
    if [ -f "$PID_FILE" ]; then
        PID="$(cat "$PID_FILE" 2>/dev/null || true)"
        if [ -n "$PID" ] && kill -0 "$PID" 2>/dev/null; then
            echo "El servidor ya se encuentra en ejecución (PID: $PID)."
            echo "URL: http://localhost:$PORT"
            return 0
        else
            rm -f "$PID_FILE"
        fi
    fi

    # Verificar Docker disponible
    if ! docker info >/dev/null 2>&1; then
        echo "Error: Docker no está en ejecución o no se tienen permisos para acceder al socket de Docker."
        exit 1
    fi

    # 1. Tareas previas: sqlc, templ, compilación, limpiar y levantar contenedores
    if command -v sqlc >/dev/null 2>&1; then
        echo "1. Generando código con sqlc..."
        sqlc generate
    else
        echo "⚠ Advertencia: sqlc no está instalado o no se encuentra en el PATH."
    fi

    if command -v templ >/dev/null 2>&1; then
        echo "2. Generando vistas con templ..."
        templ generate
    else
        echo "⚠ Advertencia: templ no está instalado o no se encuentra en el PATH."
    fi

    echo "3. Compilando el servidor Go..."
    go build -o "$BIN_FILE" .
    echo "✓ Compilación exitosa."

    echo "4. Borrando contenedores y volúmenes previos..."
    docker compose down -v --remove-orphans

    echo "5. Levantando contenedores (PostgreSQL y Redis) y esperando disponibilidad..."
    docker compose up -d db redis
    wait_for_db
    wait_for_redis

    # 2. Ejecutar la aplicación en segundo plano
    echo "6. Iniciando servidor en segundo plano..."
    nohup "$BIN_FILE" >/dev/null 2>&1 &
    PID=$!
    disown $PID 2>/dev/null || true
    echo "$PID" > "$PID_FILE"

    # Esperar confirmación de arranque
    local started=false
    for i in $(seq 1 10); do
        if ! kill -0 "$PID" 2>/dev/null; then
            echo "❌ Error: El servidor se cayó inmediatamente tras iniciar."
            rm -f "$PID_FILE"
            exit 1
        fi

        if curl -s "http://localhost:$PORT/" >/dev/null 2>&1; then
            started=true
            break
        fi
        sleep 1
    done

    if [ "$started" = true ]; then
        echo "======================================================"
        echo "✓ Servidor iniciado con éxito en segundo plano."
        echo "  - PID: $PID"
        echo "  - URL: http://localhost:$PORT"
        echo "======================================================"
    else
        echo "======================================================"
        echo "✓ Servidor ejecutándose en segundo plano (PID: $PID)."
        echo "  - URL: http://localhost:$PORT"
        echo "======================================================"
    fi
}

stop_server() {
    local pid=""
    if [ -f "$PID_FILE" ]; then
        pid="$(cat "$PID_FILE" 2>/dev/null || true)"
    fi

    # Fallback si no está el PID file pero está el proceso o el puerto ocupado
    if [ -z "$pid" ] || ! kill -0 "$pid" 2>/dev/null; then
        local fallback_pid
        fallback_pid="$(lsof -ti :"$PORT" 2>/dev/null || pgrep -f "$BIN_FILE" 2>/dev/null | head -n 1 || true)"
        if [ -n "$fallback_pid" ]; then
            pid="$fallback_pid"
            echo "Aviso: No se encontró '$PID_FILE', pero se detectó proceso $pid en puerto $PORT."
        fi
    fi

    if [ -z "$pid" ] || ! kill -0 "$pid" 2>/dev/null; then
        echo "El servidor no está en ejecución."
        rm -f "$PID_FILE"
    else
        echo "Deteniendo el servidor (PID: $pid)..."
        kill -TERM "$pid" 2>/dev/null || true

        local timeout=10
        while kill -0 "$pid" 2>/dev/null && [ $timeout -gt 0 ]; do
            sleep 1
            timeout=$((timeout - 1))
        done

        if kill -0 "$pid" 2>/dev/null; then
            echo "El servidor no respondió a SIGTERM; forzando cierre con SIGKILL..."
            kill -9 "$pid" 2>/dev/null || true
        fi

        rm -f "$PID_FILE"
        echo "✓ Servidor detenido exitosamente."
    fi

    echo "Deteniendo y borrando contenedores y volúmenes de Docker..."
    docker compose down -v --remove-orphans
    echo "✓ Servidor, contenedores y volúmenes eliminados exitosamente."
}

restart_server() {
    echo "Reiniciando servidor..."
    stop_server
    sleep 1
    start_server
}

case "$1" in
    start)
        start_server
        ;;
    stop)
        stop_server
        ;;
    restart)
        restart_server
        ;;
    *)
        echo "Uso: $0 {start|stop|restart}"
        exit 1
        ;;
esac
