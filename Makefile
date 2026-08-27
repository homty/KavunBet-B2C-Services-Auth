.PHONY: all proto proto-wallet proto-user clean

#Default target
all: proto

#Generate Go gRPC code for all services
proto: proto-wallet proto-user

proto-wallet:
	protoc --go_out=. --go_opt=paths=source_relative \
	       --go-grpc_out=. --go-grpc_opt=paths=source_relative \
	       pkg/proto/wallet/v1/wallet.proto

proto-user:
	protoc --go_out=. --go_opt=paths=source_relative \
	       --go-grpc_out=. --go-grpc_opt=paths=source_relative \
	       pkg/proto/user/v1/user.proto

clean:
	@rm -f pkg/proto/wallet/v1/*.pb.go
	@rm -f pkg/proto/user/v1/*.pb.go
