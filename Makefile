.PHONY: build build-mips build-arm64 test clean

build:
	@mkdir -p build
	go build -o build/adguard-log-viewer .

build-mips:
	@mkdir -p build
	GOOS=linux GOARCH=mipsle GOMIPS=softfloat go build -ldflags="-s -w" -o build/adguard-log-viewer-mips .

build-arm64:
	@mkdir -p build
	GOOS=linux GOARCH=arm64 go build -ldflags="-s -w" -o build/adguard-log-viewer-arm64 .

test:
	go test -count=1 ./...

clean:
	rm -rf build/
