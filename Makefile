BINARY=dim

VET_DIR := ./cmd/... ./lib/... ./server/... ./wrapper/...
DIR_SOURCES := cmd/... lib/... server/... wrapper/...

SOURCES := $(shell find $(SOURCEDIR) -name '*.go')


#VERSION=1.0.0
#BUILD_TIME=`date +%FT%T%z`

#LDFLAGS=-ldflags "-X github.com/ariejan/roll/core.Version=${VERSION} -X github.com/ariejan/roll/core.BuildTime=${BUILD_TIME}"

default: $(BINARY)

all: clean fmt lint vet test dim integration_tests docker install

$(BINARY): $(SOURCES)
	CGO_ENABLED=0 go build -a -installsuffix cgo -o $(BINARY) .

docker: $(BINARY)
	docker build -t nhurel/dim:latest .

install: $(BINARY)
	go install

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