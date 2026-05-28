# Windows 下无需安装 GCC，禁用 CGO 编译
export CGO_ENABLED=0

.PHONY: build run compile abigen clean tidy

build:
	go build -o bin/blockchain_rw .

run:
	go run . block latest

compile:
	@mkdir -p build
	solc --abi --bin contracts/SimpleStorage.sol -o build/ --overwrite

abigen: compile
	abigen --abi=build/SimpleStorage.abi --bin=build/SimpleStorage.bin --pkg=contract --out=internal/contract/store.go

tidy:
	go mod tidy

clean:
	rm -rf build/ bin/
