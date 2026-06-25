.PHONY: dev dev-api dev-web build build-web test lint docker tidy clean

# Run backend + Vite dev server together (frontend hot-reload, /api proxied to :8080)
dev:
	@echo "Backend on :8080, frontend dev server with /api proxy"
	@$(MAKE) -j2 dev-api dev-web

dev-api:
	HS_LOG_LEVEL=debug go run ./cmd/homescape

dev-web:
	cd web && npm run dev

# Build the frontend, then the single self-contained binary with assets embedded
build: build-web
	CGO_ENABLED=0 go build -ldflags="-s -w" -o homescape ./cmd/homescape

build-web:
	cd web && npm ci && npm run build

test:
	go test ./...
	cd web && npm test

lint:
	go vet ./...
	gofmt -l .

tidy:
	go mod tidy

docker:
	docker build -t homescape:dev .

clean:
	rm -f homescape
	rm -rf web/dist
