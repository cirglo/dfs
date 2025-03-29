package proto

//go:generate protoc --go_out=. --go-grpc_out=. nodeserver.proto
//go:generate protoc --go_out=. --go-grpc_out=. nameserver.proto
