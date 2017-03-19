all: test install
install:
	GOBIN=$(GOPATH)/bin GO15VENDOREXPERIMENT=1 go install gorcon-arma/*.go
test:
	GO15VENDOREXPERIMENT=1 go test -cover `glide novendor`
vet:
	go tool vet .
	go tool vet --shadow .
lint:
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
	-version=0.1.0 \
	-config=src/play-net.org/gorcon-arma/deb/create_deb_config.json

