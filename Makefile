DB_DSN?=postgres://postgres:postgres@localhost:5432/gopulse_messages?sslmode=disable

unit-test:
	go test -v -tags=unit ./...

integration-test:
	go test -v -tags=integration ./...

linter:
	golangci-lint run ./... 

linter-fix:
	golangci-lint run --fix ./...

govulncheck:
	govulncheck ./...

gosec:
	gosec ./...

pre-commit:
	make unit-test
	make integration-test
	make linter
	make gosec
	make govulncheck

migrate-create:
	migrate create -ext sql -dir migrations -seq $(name)

migrate-up:
	migrate -path migrations -database $(DB_DSN) up

migrate-down:
	migrate -path migrations -database $(DB_DSN) down
