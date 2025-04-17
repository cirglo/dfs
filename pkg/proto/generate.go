package proto

//go:generate protoc --go_out=. --go-grpc_out=. nodes.proto
//go:generate protoc --go_out=. --go-grpc_out=. names.proto
//go:generate protoc --go_out=. --go-grpc_out=. notifications.proto
//go:generate mockery --name=NotificationClient --with-expecter --dir=./ --output=../mocks --outpkg=mocks
