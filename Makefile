.PHONY: vet test run-server run-worker docker-up docker-down

vet:
	go vet ./...

test:
	go test -race -covermode=atomic ./...

run-server:
	go run ./cmd/server --db data.db --listen 8080

run-worker:
	go run ./cmd/worker --db data.db --poll 100ms

docker-up:
	docker-compose up --build --detach --wait

docker-down:
	docker-compose down