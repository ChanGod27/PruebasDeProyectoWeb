.PHONY: test clean generate

test:
	./tests/test.sh

clean:
	docker compose down -v --remove-orphans

generate:
	sqlc generate