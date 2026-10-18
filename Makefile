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

deploy: ## relay latest ghcr image into fly's registry and deploy it
	flyctl auth docker
	docker pull --platform linux/amd64 $(IMAGE):latest
	docker tag $(IMAGE):latest registry.fly.io/$(APP):latest
	docker push registry.fly.io/$(APP):latest
	flyctl deploy --app $(APP) --image registry.fly.io/$(APP):latest --ha=false
