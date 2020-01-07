SHELL=bash

DEFAULT_ENV_FILE := ~/.config/sensors/.env
# If SENSOR_ENV variable is set we use it to define environment file path
ENV_FILE := ${SENSOR_ENV}

# If env file is not set using HILO_ENV we try a default location
ifeq ($(ENV_FILE),)
	ifneq ("$(wildcard $(DEFAULT_ENV_FILE))","")
		ENV_FILE := $(DEFAULT_ENV_FILE)
	endif
endif

# If env file path is set we include it
ifneq ($(ENV_FILE),)
$(info including ENV file $(ENV_FILE))
include $(ENV_FILE)
endif

export GO111MODULE=on

LAST_TAGGED=$(shell git rev-list --tags --max-count=1)
VERSION=$(shell git describe --tags $(LAST_TAGGED))
BUILDTIME=$(shell TZ=GMT date "+%Y-%m-%d_%H:%M_GMT")
GITCOMMIT=$(shell git rev-parse --short HEAD 2>/dev/null)
GITBRANCH=$(shell git rev-parse --abbrev-ref HEAD 2>/dev/null)

COVERAGE_FILE:=coverage.out
COVERAGE_HTML:=coverage.html

SENSORS_BIN:=dist/sensors
SENSORS_SRC:=./cmd/sensors

DIST = dist

.PHONY: all
all: clean prereq test

.PHONY: prereq
prereq:
	go get golang.org/x/lint/golint
	go get golang.org/x/tools/cmd/goimports
	go mod

dirs: $(DIST)

$(DIST):
	mkdir -p $@

.PHONY: clean
clean:
	rm -rf dist

.PHONY: test
test:
	INTEG=$(INTEG) go test --covermode=count -coverprofile=$(COVERAGE_FILE) ./... && tail -q -n +2 $(COVERAGE_FILE) | go tool cover -func=$(COVERAGE_FILE)
	go tool cover -html=$(COVERAGE_FILE) -o $(COVERAGE_HTML)

.PHONY: race
race:
	go test -race -short ./...

.PHONY: msan
msan:
	go test -msan -short ./...

.PHONY: lint
lint:
	golint -set_exit_status ./...

.PHONY: fmt
fmt:
	goimports -l ./

LD_FLAGS:=-s -w -X github.com/mklimuk/sensors/cmd/sensors.version=$(VERSION) -X github.com/mklimuk/sensors/cmd/sensors.commit=$(GITCOMMIT) -X github.com/mklimuk/sensors/cmd/sensors.date=$(BUILDTIME)

.PHONY: build
compile: dirs
	GO111MODULE=on CGO_ENABLED=1 go build -ldflags '$(LD_FLAGS)' -o $(SENSORS_BIN) -v $(SENSORS_SRC)

# this is missing C cross compiler setup
.PHONY: compile-linux
compile-linux: dirs
	GOOS=linux GOARCH=amd64 go build -ldflags '$(LD_FLAGS)' -o $(SENSORS_BIN) -v $(SENSORS_SRC)





