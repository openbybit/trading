#!/bin/bash

protoc --go_out=. --go-grpc_out=. ./price_service.proto ./money.proto

# protoc --go_out=../index --go_opt=paths=source_relative \
#     --go-grpc_out=../index --go-grpc_opt=paths=source_relative \
#     ./price_service.proto ./money.proto