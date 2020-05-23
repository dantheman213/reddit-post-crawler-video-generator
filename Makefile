BIN_NAME := rpcvg
BIN_PATH := bin/$(BIN_NAME)
BUILD_FLAGS := -installsuffix "static"

ifeq ($(OS),Windows_NT)
	BIN_NAME := $(BIN_NAME).exe
	BIN_PATH := bin/$(BIN_NAME)
else
	UNAME_S := $(shell uname -s)
	ifeq ($(UNAME_S),Linux)
		# TODO
	endif
	ifeq ($(UNAME_S),Darwin)
		# https://github.com/zserge/webview/issues/13
		# minimal.app/Contents/Info.plist
		# minimal.app/Contents/Resources/minimal.icns
		PRE_PATH := bin/$(BIN_NAME).app/Contents/MacOS
		BIN_PATH := $(PRE_PATH)/$(BIN_NAME)
	endif
endif

.PHONY: all build clean deps

all: build

build:
	CGO_ENABLED=1 \
	GO111MODULE=on \
	GOARCH=amd64 \
	go build \
	$(BUILD_FLAGS) \
	-o $(BIN_PATH) \
	$$(find cmd/app/*.go)

clean:
	@echo Cleaning bin/ directory... && \
		rm -rfv bin/

deps:
	@echo Downloading go.mod dependencies && \
		go mod download
