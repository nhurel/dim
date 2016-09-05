SHELL := /bin/bash
BINARY=dim

VET_DIR := ./cmd/... ./lib/... ./server/... ./wrapper/...
DIR_SOURCES := cmd/... lib/... server/... wrapper/...

SOURCES := $(shell find $(SOURCEDIR) -name '*.go')

git_tag = $(shell git describe --tags --long | sed -e 's/-/./g' | awk -F '.' '{print $$1"."$$2"."$$3+$$4}')

#VERSION=1.0.0
#BUILD_TIME=`date +%FT%T%z`

#LDFLAGS=-ldflags "-X github.com/ariejan/roll/core.Version=${VERSION} -X github.com/ariejan/roll/core.BuildTime=${BUILD_TIME}"

default: $(BINARY)

all: clean fmt lint vet test dim integration_tests docker install

$(BINARY): $(SOURCES)
	CGO_ENABLED=0 go build -a -installsuffix cgo -o $(BINARY) -ldflags "-X main.Version=$(git_tag)" .

docker: $(BINARY)
	docker build -t nhurel/dim:latest .

install:
	go install -ldflags "-X main.Version=$(git_tag)"

.PHONY: clean install vet lint fmt

clean:
	if [ -f ${BINARY} ] ; then rm ${BINARY} ; fi

test: fmt
	go test ${VET_DIR}

vet: fmt
	go vet ${VET_DIR}
	go vet main.go

lint: fmt
	for d in $(DIR_SOURCES); do golint $$d; done
	golint main.go

fmt:
	go fmt ./...


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