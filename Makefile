
default: build

build:
	go install ./...

format:
	go fmt ./...

vet:
	go vet ./...

test:
	go test ./...

check: vet test
