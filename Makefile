all: test build

test:
	go vet ./...
	go test ./...

apitest:
	./apitest -d test/

gox:
	go get github.com/mitchellh/gox
	gox ${LDFLAGS} -parallel=4 -output="./bin/apitest_{{.OS}}_{{.Arch}}"

clean:
	rm -rfv ./apitest ./bin/*

build:
	go build

.PHONY: all test apitest gox build clean
