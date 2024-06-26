GOOS ?= linux
GOARCH ?= amd64
GIT_COMMIT_SHA ?= $(shell git rev-list -1 HEAD)
LD_FLAGS = -ldflags="-X main.buildCommit=${GIT_COMMIT_SHA}"

all: test build apitest

deps:
	go mod download github.com/clbanning/mxj
	go get ./...

vet:
	go vet ./...

fmt:
	go fmt ./...

test: deps fmt vet
	go test -race -cover ./...

webtest:
	go test -coverprofile=testcoverage.out
	go tool cover -html=testcoverage.out

apitest:
	./apitest -c apitest.test.yml --stop-on-fail -d test/

gox: deps
	go get github.com/mitchellh/gox
	gox ${LDFLAGS} -parallel=4 -output="./bin/apitest_{{.OS}}_{{.Arch}}"

clean:
	rm -rfv ./apitest ./bin/* ./testcoverage.out

ci: deps
	go build $(LD_FLAGS) -o bin/apitest_$(GOOS)_$(GOARCH) *.go

build: deps
	go build $(LD_FLAGS)

build-linux: deps
	GOOS=linux GOARCH=amd64 go build -o apitest-linux

.PHONY: all test apitest webtest gox build clean
