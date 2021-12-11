APP:=$(notdir $(patsubst %/,%, $(CURDIR)))
TAG:=$(git describe --abbrev=0 --tags)

all: test build

## test: run tests
test:
	go test ./...

## build: build binary locally
build: $(APP)
$(APP):
	go build -trimpath -ldflags "-s -w -X main.version=$(TAG)" -o $@ .

## help: print this information
help: Makefile
	echo ' Choose a command to run in $(APP):'
	sed -n 's/^##//p' $< | column -t -s ':' | sed -e 's/^/ /'

.PHONY: test build help
.SILENT: test help
