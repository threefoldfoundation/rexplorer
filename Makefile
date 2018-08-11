STANDARD_REDIS_ADDR = :6379
STANDARD_REDIS_DB = 0
TESTNET_REDIS_ADDR = :6379
TESTNET_REDIS_DB = 1

version = $(shell git describe | cut -d '-' -f 1)
commit = $(shell git rev-parse --short HEAD)
ifeq ($(commit), $(shell git rev-list -n 1 $(version) | cut -c1-7))
fullversion = $(version)
else
fullversion = $(version)-$(commit)
endif

stdbindir = $(GOPATH)/bin
ldflagsversion = -X main.rawVersion=$(fullversion)

install-std:
	go build -ldflags "$(ldflagsversion) -s -w" -o $(stdbindir)/rexplorer .

install:
	go build -race -tags "debug dev" -ldflags "$(ldflagsversion)" -o $(stdbindir)/rexplorer .

integration-tests: integration-test-sumcoins

integration-test-sumcoins:
	go run tests/integration/sumcoins/main.go --db-address "$(TESTNET_REDIS_ADDR)" --db-slot "$(TESTNET_REDIS_DB)"
	go run tests/integration/sumcoins/main.go --db-address "$(STANDARD_REDIS_ADDR)" --db-slot "$(STANDARD_REDIS_DB)"
