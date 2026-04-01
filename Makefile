.PHONY: dev db-up migrate-up gen

dev:
	go run ./cmd/shortwarden

db-up:
	docker compose up -d

migrate-up:
	go run ./cmd/migrate

gen:
	sqlc generate
	oapi-codegen --package api --generate types -o api/gen/types.gen.go api/openapi.yaml
	oapi-codegen --package api --generate chi-server -o api/gen/chi-server.gen.go api/openapi.yaml

