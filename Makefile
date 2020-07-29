all: test build

vet:
	go vet ./...

fmt:
	go fmt ./...

test: fmt vet
	go test -race -cover ./...

webtest:
	go test -coverprofile=output.out
	go tool cover -html=output.out

apitest:
	./apitest --stop-on-fail -d test/

gox:
	go get github.com/mitchellh/gox
	gox ${LDFLAGS} -parallel=4 -output="./bin/apitest_{{.OS}}_{{.Arch}}"

clean:
	rm -rfv ./apitest ./bin/* ./output.out

build:
	go build

.PHONY: all test apitest gox build clean
