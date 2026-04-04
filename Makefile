.PHONY: all build test publish clean

all: build test

build:
	go build ./...

test:
	go test -v ./...

publish:
	GOPROXY=proxy.golang.org go list -m github.com/jsalio/thunder_framework@latest

clean:
	go clean
