BACKEND  := backend
FRONTEND := frontend
IMAGE    ?= sezzle-calculator:local
PORT     ?= 8080

.DEFAULT_GOAL := help
.PHONY: help dev-backend dev-frontend test test-backend test-frontend \
        coverage coverage-backend coverage-frontend lint lint-backend lint-frontend \
        check docker-build docker-run smoke clean

help: ## List targets
	@grep -E '^[a-zA-Z_-]+:.*?## ' $(MAKEFILE_LIST) | awk 'BEGIN{FS=":.*?## "}{printf "  \033[36m%-18s\033[0m %s\n",$$1,$$2}'

dev-backend: ## Run the Go API on :8080
	cd $(BACKEND) && go run ./cmd/server

dev-frontend: ## Run Vite on :5173 (proxies /api to :8080)
	cd $(FRONTEND) && npm run dev

test-backend: ## Go tests with race detector
	cd $(BACKEND) && go test -race -count=1 ./...

test-frontend: ## Vitest (single run)
	cd $(FRONTEND) && npx vitest run

test: test-backend test-frontend ## All tests

coverage-backend: ## Go coverage report + 90% gate on internal/...
	mkdir -p coverage
	cd $(BACKEND) && go test -race -count=1 -covermode=atomic -coverprofile=../coverage/backend.out ./internal/...
	cd $(BACKEND) && go tool cover -html=../coverage/backend.out -o ../coverage/backend.html
	./scripts/check-go-coverage.sh coverage/backend.out 90

coverage-frontend: ## Vitest coverage + thresholds
	cd $(FRONTEND) && npm run coverage

coverage: coverage-backend coverage-frontend ## All coverage reports

lint-backend: ## gofmt + go vet
	@out="$$(gofmt -l $(BACKEND))"; if [ -n "$$out" ]; then echo "gofmt needed:"; echo "$$out"; exit 1; fi
	cd $(BACKEND) && go vet ./...

lint-frontend: ## eslint + tsc
	cd $(FRONTEND) && npm run lint && npm run typecheck

lint: lint-backend lint-frontend ## All linters

check: lint coverage ## Full quality gate (what CI runs)

docker-build: ## Build the full-stack image
	docker build -t $(IMAGE) .

docker-run: ## Run the image on :$(PORT)
	docker run --rm -p $(PORT):8080 --name sezzle-calculator $(IMAGE)

smoke: ## Smoke-test a running server (BASE_URL defaults to localhost:$(PORT))
	./scripts/smoke.sh $${BASE_URL:-http://localhost:$(PORT)}

clean: ## Remove build and coverage outputs
	rm -rf coverage $(FRONTEND)/coverage $(FRONTEND)/dist
