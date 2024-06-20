#!/bin/bash

PROTO_DIR="./proto"
VALIDATE_DIR="../../../proto"
OPTIONS_DIR="../../../core/grpc/proto"
OUT_DIR="."

mkdir -p $OUT_DIR

protoc --proto_path=$PROTO_DIR --proto_path=$VALIDATE_DIR --proto_path=$OPTIONS_DIR \
       --go_out=$OUT_DIR --go_opt=paths=source_relative \
       --go-grpc_out=$OUT_DIR --go-grpc_opt=paths=source_relative \
       --validate_out="lang=go:$OUT_DIR" --validate_opt=paths=source_relative \
       $PROTO_DIR/balance_service.proto

echo "Protobuf files compiled successfully!"
