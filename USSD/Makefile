.PHONY: build run test lint clean docker-build docker-run fmt tidy simulate

# --- Build ---
build:
	go build -ldflags="-s -w" -o bin/ussd-gateway cmd/server/main.go

run:
	go run cmd/server/main.go

# --- Dependencies ---
tidy:
	go mod tidy

fmt:
	gofmt -s -w .

# --- Testing ---
test:
	go test -v -race -count=1 ./...

test-cover:
	go test -race -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html

bench:
	go test -bench=. -benchmem ./...

# --- Linting ---
lint:
	golangci-lint run ./...

# --- Docker ---
docker-build:
	docker build -t amy-mis-ussd:latest .

docker-run:
	docker compose up -d

docker-stop:
	docker compose down

docker-logs:
	docker compose logs -f ussd

# --- Load Testing ---
# Simulates concurrent USSD sessions (requires curl)
simulate:
	@echo "Simulating USSD session..."
	@curl -s -X POST http://localhost:8090/ussd/simulator \
		-H "Content-Type: application/json" \
		-d '{"session_id":"test-001","phone_number":"+254712345678","service_code":"*384*123#","input":""}' | jq .
	@echo ""
	@echo "Select option 1 (Check Balance):"
	@curl -s -X POST http://localhost:8090/ussd/simulator \
		-H "Content-Type: application/json" \
		-d '{"session_id":"test-001","phone_number":"+254712345678","service_code":"*384*123#","input":"1"}' | jq .

# Stress test with concurrent sessions
stress:
	@echo "Running stress test with 100 concurrent sessions..."
	@for i in $$(seq 1 100); do \
		curl -s -X POST http://localhost:8090/ussd/simulator \
			-H "Content-Type: application/json" \
			-d "{\"session_id\":\"stress-$$i\",\"phone_number\":\"+2547123456$$i\",\"service_code\":\"*384*123#\",\"input\":\"\"}" &; \
	done; wait
	@echo "Stress test complete"

# --- Clean ---
clean:
	rm -rf bin/ coverage.out coverage.html
