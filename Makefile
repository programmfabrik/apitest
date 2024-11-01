all: test build apitest

test:
	go vet ./...
	go test -race -cover ./...

apitest: build
	./apitest -c apitest.test.yml --stop-on-fail -d test/

clean:
	rm -rfv ./apitest ./bin/* ./testcoverage.out

build:
	go build

.PHONY: all test apitest build clean
