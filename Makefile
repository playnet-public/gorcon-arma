###################### PlayNet Makefile ######################
#
# This Makefile is used to manage the PlayNet command-line template
# All possible tools have to reside under their respective folders in cmd/
# and are being autodetected.
# 'make full' would then process them all while 'make toolname' would only
# handle the specified one(s).
# Edit this file with care, as it is also being used by our CI/CD Pipeline
# For usage information check README.md
#
# Parts of this makefile are based upon github.com/kolide/kit
#

NAME         := gorcon-arma
REPO         := playnet-public
GIT_HOST     := github.com
REGISTRY     := quay.io
IMAGE        := playnet/$(NAME)

PATH := $(GOPATH)/bin:$(PATH)
TOOLS_DIR := cmd
VERSION = $(shell git describe --tags --always --dirty)
BRANCH = $(shell git rev-parse --abbrev-ref HEAD)
REVISION = $(shell git rev-parse HEAD)
REVSHORT = $(shell git rev-parse --short HEAD)
USER = $(shell whoami)

OS_PLATFORMS = linux darwin windows
OS_ARCHS = 386 amd64
MAKEFLAGS += --no-print-directory

SUBDIRS := $(patsubst $(TOOLS_DIR)/%,%,$(wildcard $(TOOLS_DIR)/*))

# if SINGLE_TOOL=1 the targets will work without specifying full/toolname
# set to != 1 if never more than one
SINGLE_TOOL := $(words $(SUBDIRS))
$(if $(findstring full,$(MAKECMDGOALS)), $(eval SINGLE_TOOL=2),)
TARGETS ?= default

-include .env

include helpers/make_version
include helpers/make_gohelpers
include helpers/make_dockerbuild
include helpers/make_db

### MAIN STEPS ###

default: .build-all

# install required tools and dependencies
deps:
	go get -u golang.org/x/tools/cmd/goimports
	go get -u github.com/golang/lint/golint
	go get -u github.com/kisielk/errcheck
	go get -u github.com/golang/dep/cmd/dep
ifdef BUILD_DEB
	go get -u github.com/bborbe/debian_utils/bin/create_debian_package
endif
	dep ensure

# test entire repo
test:
	@go test -cover -race $(shell go list ./... | grep -v /vendor/)

# install passed in tool project
install:
	$(if $(TOOL),GOBIN=$(GOPATH)/bin go install $(TOOLS_DIR)/$(TOOL)/*.go, \
	$(if $(filter-out 1,$(SINGLE_TOOL)),, GOBIN=$(GOPATH)/bin go install $(TOOLS_DIR)/$(strip $(SUBDIRS))/*.go))

# build passed in tool project
build: .pre-build
	$(if $(TOOL),GOBIN=$(GOPATH)/bin go build -i -o build/$(TOOL) -ldflags ${KIT_VERSION} ./$(TOOLS_DIR)/$(TOOL)/, \
	$(if $(filter-out 1,$(SINGLE_TOOL)),, GOBIN=$(GOPATH)/bin go build -i -o build/$(strip $(SUBDIRS)) -ldflags ${KIT_VERSION} ./$(TOOLS_DIR)/$(strip $(SUBDIRS))/))

crossbuild: .pre-build
	@for OS in $(OS_PLATFORMS); do \
		for ARCH in $(OS_ARCHS); do \
			echo building for $$OS-$$ARCH; \
			$(if $(TOOL),GOARCH=$$ARCH GOOS=$$OS CGO_ENABLED=0 GOBIN=$(GOPATH)/bin go build -o build/$(TOOL)_$$OS-$$ARCH -ldflags ${KIT_VERSION} ./$(TOOLS_DIR)/$(TOOL)/, \
			$(if $(filter-out 1,$(SINGLE_TOOL)),exit 0, GOARCH=$$ARCH GOOS=$$OS CGO_ENABLED=0 GOBIN=$(GOPATH)/bin go build -o build/$(strip $(SUBDIRS))_$$OS-$$ARCH -ldflags ${KIT_VERSION} ./$(TOOLS_DIR)/$(strip $(SUBDIRS))/)); \
		done \
	done

# execute targets for all tool projects
full:
	$(eval MAKE_TARGETS=$(strip $(subst full,,$(MAKECMDGOALS))))
	$(eval TARGETS=$(strip $(filter-out $(SUBDIRS),$(MAKE_TARGETS))))
	@for dir in $(SUBDIRS); do \
		make $$dir $(TARGETS); \
	done

# run specified tool binary
run: build
	@$(if $(TOOL),./build/$(TOOL) \
	-logtostderr \
	-v=2, \
	$(if $(filter-out 1,$(SINGLE_TOOL)),, ./build/$(strip $(SUBDIRS)) \
	-logtostderr \
	-v=2))

# run specified tool from code
dev:
	@$(if $(TOOL),go run -ldflags ${KIT_VERSION} $(TOOLS_DIR)/$(TOOL)/*.go \
	-logtostderr \
	-v=4 -debug, \
	$(if $(filter-out 1,$(SINGLE_TOOL)),, go run -ldflags ${KIT_VERSION} $(TOOLS_DIR)/$(strip $(SUBDIRS))/*.go \
	-logtostderr \
	-v=4 -debug))

# build the docker image
docker: build-in-docker 
	TOOL=$(NAME) make build-image

# upload the docker image
upload:
	docker push $(REGISTRY)/$(IMAGE)

### HELPER STEPS ###

# execute targets on single tool projects
$(SUBDIRS):
	@echo ""
	$(eval TARGETS=$(strip $(filter-out $(SUBDIRS),$(MAKECMDGOALS))))
	TOOL=$@ make $(TARGETS)

# clean local vendor folder
clean:
	rm -rf build
	docker rmi -f $(shell docker images -q --filter=reference=$(REGISTRY)/$(IMAGE)*)

build-docker-bin:
	$(if $(TOOL),GOBIN=$(GOPATH)/bin CGO_ENABLED=0 GOOS=linux go build -o build/$(TOOL) -ldflags ${KIT_VERSION_DOCKER} -a -installsuffix cgo ./$(TOOLS_DIR)/$(TOOL)/, )

.pre-build:
	@mkdir -p build

.build-all:
	make full build

update-deployment: docker upload clean restart-deployment

restart-deployment:
	kubectl delete po -n $(NAME) -lapp=$(NAME)
