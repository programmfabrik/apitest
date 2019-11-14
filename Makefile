GITCOMMIT=`git log --pretty=format:'%h' -n 1`
BUILDTIMESTAMP=`date -u +%d.%m.%Y_%H:%M:%S_%Z`
GITVERSIONTAG=`git tag -l 'v*' | tail -1`
LDFLAGS=-ldflags "-X main.buildTimeStamp=${BUILDTIMESTAMP} -X main.gitVersion=${GITVERSIONTAG} -X main.gitCommit=${GITCOMMIT}"

generate:
	go generate ./...
test: generate
	go vet ./...
	go test ./...
gox: generate
	go get github.com/mitchellh/gox
	gox ${LDFLAGS} -parallel=4 -output="./bin/apitest_{{.OS}}_{{.Arch}}" ./cmd/apitest/
build: generate
	go build ${LDFLAGS} -o ./bin/apitest ./cmd/apitest/

