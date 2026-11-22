BIN=bin/md-backend-monolith
DOCKER_IMAGE?=md-backend-monolith
WS_ROOT:=$(abspath .)
# Ignore backend/go.work — old services have conflicting pkg replaces.
export GOWORK=off

.PHONY: build test lint run docker-build atlas-hash atlas-lint atlas-validate tidy setup-db

build:
	mkdir -p bin
	go build -o $(BIN) ./cmd/server

test:
	go test ./...

lint:
	go vet ./...

tidy:
	go mod tidy

run: build
	set -a && [ -f .env ] && . ./.env; set +a; ./$(BIN)

docker-build:
	docker build -f md-backend-monolith/Dockerfile -t $(DOCKER_IMAGE) $(abspath $(WS_ROOT)/..)

atlas-hash:
	atlas migrate hash --dir file://migrations

atlas-lint:
	atlas migrate lint --env local --latest 1

atlas-validate:
	atlas migrate validate --dir file://migrations

# Requires DATABASE_URL (admin). Schema only — generates app creds and writes .env
setup-db:
	./scripts/setup-db.sh
