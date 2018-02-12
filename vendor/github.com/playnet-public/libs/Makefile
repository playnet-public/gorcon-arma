######################### PlayNet Makefile #########################
#
# This Makefile is used to manage the PlayNet Libs
# All it includes for now is testing, as nothing is being packaged
#
# Parts of this makefile are based upon github.com/kolide/kit
#

NAME         := libs
REPO         := playnet-public
GIT_HOST     := github.com
REGISTRY     := quay.io
IMAGE        := playnet/$(NAME)

PATH := $(GOPATH)/bin:$(PATH)
VERSION = $(shell git describe --tags --always --dirty)
BRANCH = $(shell git rev-parse --abbrev-ref HEAD)
REVISION = $(shell git rev-parse HEAD)
REVSHORT = $(shell git rev-parse --short HEAD)
USER = $(shell whoami)


include helpers/make_version
include helpers/make_gohelpers


### MAIN STEPS ###

default: test

# install required tools and dependencies
deps:
	go get -u golang.org/x/tools/cmd/goimports
	go get -u github.com/golang/lint/golint
	go get -u github.com/kisielk/errcheck
	go get -u github.com/golang/dep/cmd/dep
	dep ensure

# test entire repo
test:
	@go test -cover -race $(shell go list ./... | grep -v /vendor/)
