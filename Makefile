build:
	@go build -o .bin/fkt .

test: export LOG_LEVEL=trace
test: export LOG_FORMAT=console
test: export BASE_DIRECTORY=$(CURDIR)/example
test: export CONFIG_FILE=$(CURDIR)/example/config.yaml
test: build
	@echo Processing...
	@.bin/fkt
	@echo
	@echo Diffing...
	@.bin/fkt -d -l none && echo "No differences" || echo "ERROR: Differences found"

vendor: tidy
	go mod vendor

tidy:
	go mod tidy

clean:
	rm .bin/*
	rm -rf example/overlays

.PHONY: build test vendor tidy clean