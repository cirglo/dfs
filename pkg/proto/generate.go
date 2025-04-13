package proto

//go:generate protoc --go_out=. --go-grpc_out=. nodeserver.proto
//go:generate protoc --go_out=. --go-grpc_out=. nameserver.proto
//go:generate protoc --go_out=. --go-grpc_out=. internal.proto
//go:generate mockery --name=NameInternalClient --with-expecter --dir=./ --output=../mocks --outpkg=mocks
