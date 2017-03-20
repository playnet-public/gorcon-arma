.PHONY: all

GORCON_ENVS := \
	-e OS_ARCH_ARG \
	-e OS_PLATFORM_ARG \
	-e TESTFLAGS \
	-e VERBOSE \
	-e VERSION

BIND_DIR := "dist"

GIT_BRANCH := $(subst heads/,,$(shell git rev-parse --abbrev-ref HEAD 2>/dev/null))
REPONAME := $(shell echo $(REPO) | tr '[:upper:]' '[:lower:]')

print-%: ; @echo $*=$($*)

default: binary

all: ## validate all checks, build linux binary, run all tests\ncross non-linux binaries
	./script/make.sh

binary:
	./script/make.sh generate binary

crossbinary: ## cross build the non-linux binaries
	./script/make.sh generate crossbinary

test: ## run the unit and integration tests
	./script/make.sh generate test-unit binary test-integration

test-unit: ## run the unit tests
	./script/make.sh generate test-unit

test-integration: ## run the integration tests
	./script/make.sh generate binary test-integration

validate: ## validate gofmt, golint and go vet
	./script/make.sh  validate-glide validate-gofmt validate-govet validate-golint validate-misspell validate-vendor

dist:
	mkdir dist
	
all: test install
install:
	GOBIN=$(GOPATH)/bin GO15VENDOREXPERIMENT=1 go install gorcon-arma/*.go
test:
	GO15VENDOREXPERIMENT=1 go test -cover `glide novendor`
vet:
	go tool vet .
	go tool vet --shadow .
lint:
	script/validate-golint
	golint -min_confidence 1 ./...
errcheck:
	errcheck -ignore '(Close|Write|SetReadDeadline|SetWriteDeadline)' ./...
check: lint vet errcheck
format:
	find . -name "*.go" -exec gofmt -w "{}" \;
	goimports -w=true .
prepare:
	go get -u golang.org/x/tools/cmd/goimports
	go get -u github.com/Masterminds/glide
	go get -u github.com/golang/lint/golint
	go get -u github.com/kisielk/errcheck
	go get -u github.com/bborbe/debian_utils/bin/create_debian_package
	glide install
update:
	glide up
clean:
	rm -rf var vendor target
package:
	cd $(GOPATH) && create_debian_package \
	-logtostderr \
	-v=2 \
	-version=$(VERSION) \
	-config=src/play-net.org/gorcon-arma/deb/create_deb_config.json


