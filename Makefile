ifndef GOPATH
$(error You need to set up a GOPATH.  See the README file.)
endif

PATH = $(GOPATH)/bin:$(shell printenv PATH)

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
