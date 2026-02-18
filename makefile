.PHONY: build run test migrate docker-up docker-down swagger

build:
	go build -o bin/subscription-service ./cmd/main.go

run:
	go run cmd/main.go

test:
	go test -v ./...

migrate-up:
	migrate -path migrations -database "postgresql://postgres:postgres@localhost:5432/subscriptions?sslmode=disable" up

migrate-down:
	migrate -path migrations -database "postgresql://postgres:postgres@localhost:5432/subscriptions?sslmode=disable" down

docker-up:
	docker-compose up -d --build

docker-down:
	docker-compose down -v

swagger:
	swag init -g cmd/main.go

clean:
	rm -rf bin/