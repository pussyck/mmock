.PHONY: build doc fmt lint dev test vet docker docker-build docker-run docker-stop docker-rm docker-restart

PKG_NAME=mmock
NS = jordimartin
VERSION ?= latest
DOCKER_IMAGE = $(PKG_NAME):$(VERSION)
CONTAINER_NAME = mmock

export GO111MODULE=on


build: vet \
	test
	npm --prefix UI run build
	go build  -v -o ./bin/$(PKG_NAME) cmd/mmock/main.go

doc:
	godoc -http=:6060

fmt:
	go fmt ./...

# https://github.com/golang/lint
# go get github.com/golang/lint/golint
lint:
	golint ./...

test:
	go test -v ./...
	
coverage:
	./coverage.sh

# https://godoc.org/golang.org/x/tools/cmd/vet
vet:
	go vet -v  ./...

release:
	goreleaser --clean

# Build Linux binary for Docker
build-linux:
	@echo "Building Linux binary..."
	GOOS=linux GOARCH=amd64 go build -v -o ./mmock cmd/mmock/main.go

# Install npm dependencies if needed
npm-install:
	@if [ ! -d "UI/node_modules" ]; then \
		echo "Installing npm dependencies..."; \
		npm --prefix UI install; \
	fi

# Build UI
build-ui: npm-install
	@echo "Building UI..."
	npm --prefix UI run build



up: 
	$(MAKE) docker-build
	$(MAKE) docker-run

down: 
	$(MAKE) docker-stop
	$(MAKE) docker-rm
	@echo "Stopping and removing Docker container..."

restart: 
	$(MAKE) down
	$(MAKE) up

# Build Docker image (includes all prerequisites)
docker-build: build-ui build-linux
	@echo "Building Docker image..."
	docker build -t $(DOCKER_IMAGE) .
	@echo "Docker image $(DOCKER_IMAGE) built successfully!"

# Alias for docker-build
docker: docker-build

# Run Docker container
docker-run:
	@echo "Starting Docker container..."
	docker run -d --name $(CONTAINER_NAME) \
		-v "$(shell pwd)/config:/config" \
		-p 8082:8082 \
		-p 8083:8083 \
		-p 8084:8084 \
		$(DOCKER_IMAGE)
	@echo "Container $(CONTAINER_NAME) started!"
	@echo "Web UI: http://localhost:8082"
	@echo "Mock server: http://localhost:8083"

# Stop Docker container
docker-stop:
	@echo "Stopping Docker container..."
	docker stop $(CONTAINER_NAME) || true

# Remove Docker container
docker-rm: docker-stop
	@echo "Removing Docker container..."
	docker rm $(CONTAINER_NAME) || true

# Restart Docker container
docker-restart: docker-rm docker-run

docker-push:
	docker build --no-cache=true  -t $(NS)/$(PKG_NAME):$(VERSION) .
	docker push $(NS)/$(PKG_NAME):$(VERSION)
