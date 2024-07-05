package main

// PROTO_DIR="./proto"
// VALIDATE_DIR="../../../proto"
// OPTIONS_DIR="../../../core/routing/grpc/proto"
// OUT_DIR="./service"

// mkdir -p $OUT_DIR

// protoc --proto_path=$PROTO_DIR --proto_path=$VALIDATE_DIR --proto_path=$OPTIONS_DIR \
//        --go_out=$OUT_DIR --go_opt=paths=source_relative \
//        --go-grpc_out=$OUT_DIR --go-grpc_opt=paths=source_relative \
//        --validate_out="lang=go:$OUT_DIR" --validate_opt=paths=source_relative \
//        $PROTO_DIR/balance_service.proto

//go:generate mkdir -p ./service

//go:generate protoc --proto_path=./proto --proto_path=../../../proto --proto_path=../../../core/routing/grpc/proto --go_out=./service --go_opt=paths=source_relative --go-grpc_out=./service --go-grpc_opt=paths=source_relative --validate_out=lang=go:./service --validate_opt=paths=source_relative ./proto/balance_service.proto
