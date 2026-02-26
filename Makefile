VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
BIN := bin/csb

.PHONY: build run install clean tidy

build:
	go build -ldflags "-X main.version=$(VERSION)" -o $(BIN) .

run: build
	$(BIN)

install: build
	cp $(BIN) ~/.local/bin/csb

clean:
	rm -rf bin/

tidy:
	go mod tidy
