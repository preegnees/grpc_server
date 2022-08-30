.SILENT:

serv:
	go run cmd/server/main.go

cli:
	go run cmd/client/main.go

proto:
	protoc --go_out=. --go_opt=paths=source_relative --go-grpc_out=. --go-grpc_opt=paths=source_relative pkg/proto/service.proto