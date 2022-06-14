
REVISION_VAR := main.commitHash
REVISION_VALUE ?= $(shell git rev-parse --short HEAD 2>/dev/null)
BUILT_VAR := main.buildTime
BUILT_VALUE := $(shell date -u '+%Y-%m-%dT%I:%M:%S%z')

GOBUILD_LDFLAGS ?= \
	-X '$(REVISION_VAR)=$(REVISION_VALUE)' \
	-X '$(BUILT_VAR)=$(BUILT_VALUE)'

export GO111MODULE=on

.PHONY: test coverage clean build release

test:
	go test -v -race

coverage:
	go test -covermode=atomic -coverprofile=coverage.out

build:
	(cd cmd/ && go build -o ../overseer -ldflags "$(GOBUILD_LDFLAGS)")

release:
	cd cmd/
	GOOS=darwin GOARCH=amd64 go build -o overseer-darwin -ldflags "-s -w $(GOBUILD_LDFLAGS)"
	GOOS=linux GOARCH=amd64 go build -o overseer-linux -ldflags "-s -w $(GOBUILD_LDFLAGS)"
	cd ..

clean:
	go clean -x -i ./...
