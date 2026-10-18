APP := whatdowedo
IMAGE := ghcr.io/ravio1i/$(APP)

.PHONY: run test build docker docker-run deploy

run: ## run locally with .env
	@set -a; [ -f .env ] && . ./.env; set +a; go run ./cmd/server

test:
	go vet ./... && go test -race ./...

build:
	CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o bin/$(APP) ./cmd/server

docker:
	docker build -t $(IMAGE):dev .

docker-run: docker
	docker run --rm -p 8080:8080 --env-file .env $(IMAGE):dev

deploy: ## deploy the latest pushed image to fly.io
	flyctl deploy --image $(IMAGE):latest
