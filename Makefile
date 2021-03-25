all: test build

vet:
	go vet ./...

fmt:
	go fmt ./...

test: fmt vet
	go test -race -cover ./...

webtest:
	go test -coverprofile=testcoverage.out
	go tool cover -html=testcoverage.out

apitest:
	./apitest --stop-on-fail -d test/

gox:
	go get github.com/mitchellh/gox
	gox ${LDFLAGS} -parallel=4 -output="./bin/apitest_{{.OS}}_{{.Arch}}"

clean:
	rm -rfv ./apitest ./bin/* ./testcoverage.out

build:
	go build

.PHONY: all test apitest webtest gox build clean
