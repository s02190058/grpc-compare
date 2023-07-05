.PHONY: gen
gen:
	protoc --go_out=. --go_opt=paths=source_relative \
	--go-grpc_out=. --go-grpc_opt=paths=source_relative \
	proto/service.proto

.PHONY: tidy
tidy:
	go mod tidy

.PHONY: run
run:
	go run ./main.go --port=9090

.PHONY: bench
bench:
	go test -v -bench=.