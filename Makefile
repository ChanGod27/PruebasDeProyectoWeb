.PHONY: test clean generate start stop stop-all restart status logs

test:
	./tests/test.sh

clean:
	docker compose down -v --remove-orphans

generate:
	sqlc generate

start: generate
	./tests/server.sh start

stop:
	./tests/server.sh stop

stop-all:
	./tests/server.sh stop-all

restart:
	./tests/server.sh restart

status:
	./tests/server.sh status

logs:
	./tests/server.sh logs
