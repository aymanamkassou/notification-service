run:
go run ./cmd/api

worker:
go run ./cmd/worker

migrate-create:
@name?=change
@ts=$$(date +%Y%m%d%H%M%S); mkdir -p db/migrations && touch db/migrations/$${ts}_$${name}.sql && echo "Created db/migrations/$${ts}_$${name}.sql"
