# メタ 情報
NAME := jsongun
VERSION := $(shell git describe --tags --abbrev=0)
REVISION := $(shell git rev-parse --short HEAD)
LDFLAGS := -X 'main.version=$(VERSION)' \
		-X 'main.revision=$(REVISON)'

## Setup
setup:
	go get github.com/Masterminds/glide
	go get github.com/golang/lint/golint
	go get golang.org/x/tools/cmd/goimports
	go get github.com/Songmu/make2help/cmd/make2help

## Test
test: deps
	go test $$(glide novendor)

## Install dependencies
deps: setup
	glide install

## Update dependencies
udpate: setup
	glide update

## Lint
lint: setup
	go vet $$(glide novendor)
	for pkg in $$(glide novendor -x); do \
		golint -set_exit_status $$pkg || exit $$?; \
	done

## Format source codes
fmt: setup
	goimports -w $$(glide nv -x)

## Build binaries
bin/%: cmd/%/main.go deps
	go build -ldflags "$(LDFLAGS)" -o $@ $<

## Show help
help:
	@make2help $(MAKEFILE_LIST)

.PHONY: setup deps update test lint help