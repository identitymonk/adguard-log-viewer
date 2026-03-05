.PHONY: build build-mips test clean

build:
	go build -o adguard-log-viewer .

build-mips:
	GOOS=linux GOARCH=mipsle GOMIPS=softfloat go build -ldflags="-s -w" -o adguard-log-viewer-mips .

test:
	go test -count=1 ./...

clean:
	rm -f adguard-log-viewer adguard-log-viewer-mips
