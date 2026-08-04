include .envrc

#==============================================================================#
# helpers
#==============================================================================#


.PHONY: help
help:
	@echo 'usage:'
	@sed -n 's/^##//p' ${MAKEFILE_LIST} | column -t -s ':' | sed -e 's/^/ /'

.PHONY: confirm
confirm:
	@echo -n 'are you sure? [y/N] ' && read ans && [ $${ans:-N} = y ]


#==============================================================================#
# development
#==============================================================================#


## run/api: run the cmd/api application
.PHONY: run/web
run/web:
	go run ./cmd/web -db-dsn=${TUCK_DB_DSN}

## db/sqlite3: connect to the database using sqlite3
.PHONY: db/sqlite3
db/sqlite3:
	sqlite3 data/tuck.db

## db/migrations/new name=$1: create a new database migration
.PHONY: db/migrations/new
db/migrations/new:
	migrate create -seq -ext=.sql -dir=./migrations ${name}

## db/migrations/up: apply all up database migrations
.PHONY: db/migrations/up
db/migrations/up: confirm
	migrate -path=./migrations -database='sqlite://data/tuck.db' up


#==============================================================================#
# quality control
#==============================================================================#


## tidy: tidy module dependencies, and format and modernize all .go files
.PHONY: tidy
tidy:
	go mod tidy
	go mod verify
	go mod vendor
	go fix ./...
	go fmt ./...


## audit: run quality control checks
.PHONY: audit
audit:
	go mod tidy -diff
	go mod verify
	go vet ./...
	go tool staticcheck ./...
	go test -race -vet=off ./...


#==============================================================================#
# build
#==============================================================================#


## build/api: build the cmd/api application
.PHONY: build/api
build/api:
	go build -ldflags='-s' -o=./bin/api ./cmd/api
	GOOS=linux GOARCH=amd64 go build -ldflags='-s' -o=./bin/linux_amd64/api ./cmd/api
