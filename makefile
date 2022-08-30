.SILENT:

serv:
	go run cmd/server/main.go

cli1:
	go run cmd/client1/main.go

cli2:
	go run cmd/client2/main.go

proto:
	protoc --go_out=. --go_opt=paths=source_relative --go-grpc_out=. --go-grpc_opt=paths=source_relative pkg/proto/service.proto