APP?=$(shell basename ${CURDIR})
BUILDDATE=$(shell date +'%Y-%m-%dT%H:%M:%SZ')
VERSION=$(shell awk -F\" '$$1~/appVersion:/{print $$2}' charts/hc/Chart.yaml)
LONGVER=${VERSION}@${BUILDDATE}

LDFLAGS=-ldflags "-s -w -X main.version=${LONGVER}"

.DEFAULT_GOAL:=default

## cover: runs go test -cover with default values
.PHONY: cover
cover:
	go test -cover ./...

## test: runs go test with default values
.PHONY: test
test:
	go test ./...

## vet: runs go vet
.PHONY: vet
vet:
	go vet ./...

## install: install to /usr/local/bin/
.PHONY: install
install: default
	mv -f ${APP} /usr/local/bin/

## default: build secgw app
.PHONY: default
default:
	@echo "Building ${APP}-${VERSION}"
	go build -o ${APP} ${LDFLAGS}

## pkg: build and package the app
.PHONY: pkg
pkg: image
	@echo "Saving ${APP}-${VERSION} image"
	docker save ${APP}:${VERSION} | gzip > ${APP}-${VERSION}-img.tgz
	
## clean: cleans the build results
.PHONY: clean
clean:
	go clean
	rm -rf *.tgz  ${APP}

## image: build docker image
.PHONY: image
image:
	GOOS=linux GOARCH=amd64 go build -o ${APP} ${LDFLAGS}
	docker build -t ${APP}:${VERSION} -f Dockerfile .

## help: prints this help message
.PHONY: help
help:
	@echo "Usage: \n"
	@sed -n 's/^##//p' ${MAKEFILE_LIST} | column -t -s ':' |  sed -e 's/^/ /'
