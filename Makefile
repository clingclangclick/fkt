build:
	@go build -o .bin/fkt .

race test validate: export LOG_LEVEL=trace
race test validate: export LOG_FORMAT=console
race test validate: export BASE_DIRECTORY=$(CURDIR)/example
race test validate: export CONFIG_FILE=$(CURDIR)/example/config.yaml
race: export GORACE="history_size=8"
race: export BIN=go run -race .
test: export BIN=.bin/fkt
test: build
race test:
	@echo Processing...
	@$(BIN)
	@echo
	@echo Diffing...
	@$(BIN) -d -l none && echo "No differences" || echo "ERROR: Differences found"

validate: export BIN=.bin/fkt
validate: build
validate:
	@echo Validating...
	@$(BIN) -v -l none

vendor: tidy
	go mod vendor

tidy:
	go mod tidy

clean:
	rm .bin/*
	rm -rf example/overlays

.PHONY: build clean race test tidy vendor