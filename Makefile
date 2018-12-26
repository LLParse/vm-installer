.PHONY: clean build install deps update-deps lint vet test dist-clean dist release tag-release

default: build

VERSION := $(shell cat VERSION)
REPO := rancher/vm-installer

mkfile_path := $(abspath $(lastword $(MAKEFILE_LIST)))
name := vm-installer

clean:
	rm -rf bin
	rm -f $(GOPATH)/bin/$(name)

build:
	CGOENABLED=0 go build -o bin/$(name)

install: build
	cp bin/$(name) $(GOPATH)/bin/

deps:
	go get -v github.com/tcnksm/ghr
	go get -v github.com/golang/lint/golint
	go get github.com/Masterminds/glide

update-deps:
	glide up

lint:
	@golint $$(go list ./... 2> /dev/null | grep -v /vendor/)

vet:
	@go vet $$(go list ./... 2> /dev/null | grep -v /vendor/)

test: lint vet
	@go test $$(go list ./... 2> /dev/null | grep -v /vendor/)

tag-release:
	git tag -f $(VERSION)
	git push -f origin master --tags

dist-clean:
	rm -rf release

dist: dist-clean
	GOOS=linux GOARCH=amd64 CGOENABLED=0 go build -o release/$(name)
	docker build -t $(REPO):$(VERSION) .

release:
	docker push $(REPO):$(VERSION)
