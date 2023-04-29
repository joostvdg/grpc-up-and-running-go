protoc:
	protoc proto/product_info.proto \
 	  -I proto \
	  --go_out=.

compile-server:
	protoc proto/*.proto \
	--go_out=./productinfo/service/ecommerce \
	--go-grpc_out=require_unimplemented_servers=false:./productinfo/service/ecommerce \
	--go_opt=paths=source_relative \
	--go-grpc_opt=paths=source_relative \
	--proto_path=proto

compile-client:
	protoc proto/*.proto \
	--go_out=./productinfo/client/ecommerce \
	--go-grpc_out=require_unimplemented_servers=false:./productinfo/client/ecommerce \
	--go_opt=paths=source_relative \
	--go-grpc_opt=paths=source_relative \
	--proto_path=proto

# require_unimplemented_servers=false
# comes from here: https://stackoverflow.com/questions/65079032/grpc-with-mustembedunimplemented-method

# This I found to work, got it form here: https://stackoverflow.com/a/72774649/5510158
# protoc:
# 	protoc ./ecommerce/product_info.proto \
# 		--proto_path=./ecommerce \
# 		--go_out=./ecommerce \
# 		--go_opt=Mproduct_info.proto=product_info/service/ecommerce 

# This is from the book, but it doesn't work for me
# protoc:
# 	protoc -I ecommerce \
# 		ecommerce/product_info.proto \
# 		--go_out=plugins=grpc:<module_dir_path>/ecommerce

run-server:
	go run ./productinfo/service/main.go

run-client:
	go run ./productinfo/client/main.go