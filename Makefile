gogen:
	protoc --proto_path=proto proto/*.proto --go_out=plugins=grpc:pb --grpc-gateway_out=:pb
jsgen:
	protoc -I=. proto/*.proto   --js_out=import_style=commonjs:.   --grpc-web_out=import_style=commonjs,mode=grpcwebtext:.
test:
	go test tests/unit_test.go
testv:
	go test -v tests/unit_test.go
