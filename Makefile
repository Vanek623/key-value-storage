.PHONY: run-server run-client
run-server:
	go run cmd/server/main.go run

run-client:
	go run cmd/client/main.go