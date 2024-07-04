package grpc

//go:generate protoc --proto_path=./proto --go_out=. --go_opt=paths=source_relative ./proto//method_options.proto
