.PHONY: server
server:
	@echo "Start server."
	@go run cmd/server/server.go

.PHONY: client
client:
	@echo "Connection server."
	@go run cmd/client/client.go
