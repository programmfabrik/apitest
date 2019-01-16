export GO111MODULE=on

SRCDIRS = .
GOFILES = $(shell find $(SRCDIRS) -name "*.go" -type f)
FYLR = fylr-apitest

GITCOMMIT=`git log --pretty=format:'%h' -n 1`
BUILDTIMESTAMP=`date -u +%d.%m.%Y_%H:%M:%S_%Z`
GITVERSIONTAG=`git tag -l 'v*' | tail -1`

# Setup the LDFlags for building the version into the binary
LDFLAGS=-ldflags "-X github.com/programmfabrik/fylr-apitest/commands.buildTimeStamp=${BUILDTIMESTAMP} -X github.com/programmfabrik/fylr-apitest/commands.gitVersion=${GITVERSIONTAG} -X github.com/programmfabrik/fylr-apitest/commands.gitCommit=${GITCOMMIT}"

all: | code
code: ensure
	go build ${LDFLAGS} -o $(FYLR)

gox: ensure
	gox ${LDFLAGS} -output="dist/fylr_{{.OS}}_{{.Arch}}"

clean:
	rm -f $(FYLR)

wipe: clean
	find $(SRCDIRS) -name '*~' -type f -exec rm '{}' \;
	find $(SRCDIRS) -name '*.sw*' -type f -exec rm '{}' \;

test: ensure $(GOBINDATA) bindata
	# add more test directories
	go clean -testcache
	go test $(TFLAGS)  ./...

fmt:
	go fmt ./...

ensure:
	go mod download

vet:
	go vet ./...

getall:
	go list -f '{{ join .Deps "\n"}}' | grep github | xargs go get

.PHONY: fmt code clean wipe all ensure vet
