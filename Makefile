SHELL := /bin/bash
BINARY=dim

VET_DIR := $(shell find . -maxdepth 1 -type d | grep -Ev "(^\./\.|\./vendor|\./dist|\./tests|^\.$$)" | sed  -e 's,.*,&/...,g' )
TEST_DIR := $(shell find . -maxdepth 1 -type d | grep -Ev "(^\./\.|\./vendor|\./dist|\./tests|\./integration|^\.$$)" | sed  -e 's,.*,&/...,g' )
DIR_SOURCES :=  $(shell find . -maxdepth 1 -type d | grep -Ev "(^\./\.|\./vendor|\./dist|\./tests|^\.$$)" | sed  -e 's,\./\(.*\),\1/...,g')
GOIMPORTS_SOURCES := $(shell find . -maxdepth 1 -type d | grep -Ev "(^\./\.|\./vendor|\./dist|\./tests|^\.$$)" | sed  -e 's,\./\(.*\),\1/,g')

SOURCES := $(shell find $(SOURCEDIR) -name '*.go')

git_tag = $(shell git describe --tags --long | sed -e 's/-/./g' | awk -F '.' '{print $$1"."$$2"."$$3+$$4}')

default: $(BINARY)

all: clean fmt lint vet test dim integration_tests docker install

$(BINARY): $(SOURCES)
	CGO_ENABLED=0 go build -a -installsuffix cgo -o $(BINARY) -ldflags "-s -X main.Version=$(git_tag)" .

distribution:
	rm -rf dist && mkdir -p dist
	docker run --rm -v "$$PWD":/go/src/github.com/nhurel/dim -w /go/src/github.com/nhurel/dim -e GOOS=windows -e GOARCH=amd64 golang:1.8.1 go build -o dist/$(BINARY)-windows.exe -ldflags "-s -X main.Version=$(git_tag)"
	docker run --rm -v "$$PWD":/go/src/github.com/nhurel/dim -w /go/src/github.com/nhurel/dim -e GOOS=linux -e GOARCH=amd64 golang:1.8.1 go build -o dist/$(BINARY)-linux-x64 -ldflags "-s -X main.Version=$(git_tag)"
	docker run --rm -v "$$PWD":/go/src/github.com/nhurel/dim -w /go/src/github.com/nhurel/dim -e GOOS=darwin -e GOARCH=amd64 golang:1.8.1 go build -o dist/$(BINARY)-darwin -ldflags "-s -X main.Version=$(git_tag)"

docker: $(BINARY)
	docker build -t nhurel/dim:$(git_tag) .

install:
	go clean -i
	go install -ldflags "-s -X main.Version=$(git_tag)"

.PHONY: clean install vet lint fmt

clean:
	if [ -f ${BINARY} ] ; then rm ${BINARY} ; fi

test: fmt
	go test ${TEST_DIR}

vet: fmt
	go vet ${VET_DIR}
	go vet main.go

lint: fmt
	for d in $(DIR_SOURCES); do golint $$d; done
	golint main.go

fmt:
	goimports -w ${GOIMPORTS_SOURCES}
	go fmt ${VET_DIR}


completion:
	go run main.go autocomplete
	sudo mv dim_compl /etc/bash_completion.d/dim_compl
	@@echo "run source ~/.bashrc to refresh completion"

integration_tests: $(BINARY)
	docker-compose up -d --build
	go test ./integration/...
	docker-compose stop && docker-compose rm -fv

current_version:
	@echo $(git_tag)

version_bump:
	git pull --tags
	n=$$(git describe --tags --long | sed -e 's/-/./g' | awk -F '.' '{print $$4}'); \
	maj=$$(git log --format=oneline -n $$n | grep "#major"); \
	min=$$(git log --format=oneline -n $$n | grep "#minor"); \
	if [ -n "$$maj" ]; then \
		TAG=$(shell git describe --tags --long | sed -e 's/-/./g' | awk -F '.' '{print $$1+1".0.0"}'); \
	elif [ -n "$$min" ]; then \
		TAG=$(shell git describe --tags --long | sed -e 's/-/./g' | awk -F '.' '{print $$1"."$$2+1".0"}'); \
	else \
		TAG=$(shell git describe --tags --long | sed -e 's/-/./g' | awk -F '.' '{print $$1"."$$2"."$$3+$$4+1}'); \
	fi; \
	git tag -a -m "Automatic version bump" $$TAG
	git push --tags
