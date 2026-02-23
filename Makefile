.PHONY: build run test lint fmt clean sqlc seed seed-demo docker docker-web migrate-runbooks

build:
	go build -o bin/bookowl ./cmd/bookowl

run:
	BOOKOWL_MODE=api \
	BOOKOWL_DB_URL=postgres://bookowl:bookowl@localhost:5433/bookowl?sslmode=disable \
	BOOKOWL_REDIS_URL=redis://localhost:6380/1 \
	BOOKOWL_DEV_MODE=true \
	go run ./cmd/bookowl

seed:
	BOOKOWL_MODE=seed \
	BOOKOWL_DB_URL=postgres://bookowl:bookowl@localhost:5433/bookowl?sslmode=disable \
	go run ./cmd/bookowl

seed-demo:
	BOOKOWL_MODE=seed-demo \
	BOOKOWL_DB_URL=postgres://bookowl:bookowl@localhost:5433/bookowl?sslmode=disable \
	go run ./cmd/bookowl

test:
	go test ./... -count=1

lint:
	golangci-lint run ./...

fmt:
	gofmt -w .

sqlc:
	sqlc generate

clean:
	rm -rf bin/

docker:
	docker build -t bookowl:dev .

docker-web:
	docker build -t bookowl-web:dev ./web

migrate-runbooks:
	BOOKOWL_MODE=migrate-nightowl-runbooks \
	BOOKOWL_DB_URL=postgres://bookowl:bookowl@localhost:5433/bookowl?sslmode=disable \
	BOOKOWL_NIGHTOWL_API_URL=http://localhost:8080 \
	BOOKOWL_NIGHTOWL_API_KEY=ow_dev_seed_key_do_not_use_in_production \
	go run ./cmd/bookowl
