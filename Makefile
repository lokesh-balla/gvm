ldflags = '-X main.version=$(version) -X main.build=$(shell date "+%Y%m%d") -extldflags "-static" -s -w'

all: clean compile-darwin-arm64 compile-darwin-amd64 compile-linux-arm64 compile-linux-amd64 compile-freebsd-arm64 compile-freebsd-amd64

compile-darwin-arm64:
	CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 go build -o ./bin/gvm_darwin_arm64 --tags netgo -ldflags $(ldflags) main.go

compile-darwin-amd64:
	CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 go build -o ./bin/gvm_darwin_amd64 --tags netgo -ldflags $(ldflags) main.go

compile-linux-arm64:
	CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -o ./bin/gvm_linux_arm64 --tags netgo -ldflags $(ldflags) main.go

compile-linux-amd64:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o ./bin/gvm_linux_amd64 --tags netgo -ldflags $(ldflags) main.go

compile-freebsd-arm64:
	CGO_ENABLED=0 GOOS=freebsd GOARCH=arm64 go build -o ./bin/gvm_freebsd_arm64 --tags netgo -ldflags $(ldflags) main.go

compile-freebsd-amd64:
	CGO_ENABLED=0 GOOS=freebsd GOARCH=amd64 go build -o ./bin/gvm_freebsd_amd64 --tags netgo -ldflags $(ldflags) main.go

vuln:
	go run golang.org/x/vuln/cmd/govulncheck@latest ./...

vet:
	go run github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.6.2 run --fix ./...

clean:
	if [ -d "./bin" ]; then rm -rf ./bin; fi
