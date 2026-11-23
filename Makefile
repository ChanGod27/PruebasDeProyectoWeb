.PHONY: generate templ start stop restart

generate:
	sqlc generate

templ:
	templ generate

start:
	./servidor.sh start

stop:
	./servidor.sh stop

restart:
	./servidor.sh restart
