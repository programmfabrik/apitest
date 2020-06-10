all: test build

test:
	go vet ./...
	go test ./...

gox:
	go get github.com/mitchellh/gox
	gox ${LDFLAGS} -parallel=4 -output="./bin/apitest_{{.OS}}_{{.Arch}}"

clean:
	rm -rfv ./apitest ./bin/*

build:
	go build

.PHONY: all test gox build clean
