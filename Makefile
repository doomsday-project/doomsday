BUILD_TARGET ?= cmd/*.go
APP_NAME := doomsday
OUTPUT_NAME ?= $(APP_NAME)
SHELL := /bin/bash
COMMIT_HASH := $(shell git log --pretty='format:%h' -n 1)
DIRTY_LINE := $(shell git diff --shortstat 2> /dev/null | tail -n1)
ifneq ("$(DIRTY_LINE)", "")
  DIRTY := +
endif
VERSION ?= development
LOCAL_GOOS=$(shell go env GOOS)
LOCAL_GOARCH=$(shell go env GOARCH)
LDFLAGS := -X "github.com/doomsday-project/doomsday/version.Version=$(VERSION)-$(COMMIT_HASH)$(DIRTY)"
BUILD := go build -v -ldflags='$(LDFLAGS)' -o $(OUTPUT_NAME) $(BUILD_TARGET)

.PHONY: build darwin darwin-amd64 darwin-arm64 linux all clean embed server tsc
.DEFAULT: build


#: Generic server build for all platfomrs
build: embed server

server: 
	@echo $(VERSION)-$(COMMIT_HASH)$(DIRTY)
	GOOS=$(GOOS) GOARCH=$(GOARCH) $(BUILD)

#: Builds all OSX executable
darwin: darwin-amd64 darwin-arm64

#: Builds arm64 OSX executable
darwin-arm64:
	GOOS=darwin GOARCH=arm64 OUTPUT_NAME=$(APP_NAME)-darwin-arm64 VERSION="$(VERSION)" $(MAKE) server

#: Builds amd64 OSX executable
darwin-amd64:
	GOOS=darwin GOARCH=amd64 OUTPUT_NAME=$(APP_NAME)-darwin-amd64 VERSION="$(VERSION)" $(MAKE) server

#: Builds amd64 linux executable
linux:
	GOOS=linux GOARCH=amd64 OUTPUT_NAME=$(APP_NAME)-linux VERSION="$(VERSION)" $(MAKE) server

#: Build client and server doomsday components
all: embed darwin linux server

embed: tsc
	GOOS="" GOARCH="" go run web/embed/main.go web/embed/mappings.yml

tsc:
	cd web && npm install
	tsc --project web/tsconfig.json

clean:
	rm -f $(APP_NAME) $(APP_NAME)-darwin-* $(APP_NAME)-linux
