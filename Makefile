STANDARD_REDIS_PORT ?= 6379
STANDARD_REDIS_ADDR ?= :$(STANDARD_REDIS_PORT)
STANDARD_REDIS_DB ?= 0
STANDARD_ENCODING_TYPE ?= msgp
TESTNET_REDIS_PORT ?= 6379
TESTNET_REDIS_ADDR ?= :$(TESTNET_REDIS_PORT)
TESTNET_REDIS_DB ?= 1
TESTNET_ENCODING_TYPE ?= msgp

version = $(shell git describe --abbrev=0)
commit = $(shell git rev-parse --short HEAD)
ifeq ($(commit), $(shell git rev-list -n 1 $(version) | cut -c1-7))
fullversion = $(version)
else
fullversion = $(version)-$(commit)
endif

stdbindir = $(GOPATH)/bin
ldflagsversion = -X main.rawVersion=$(fullversion)

testpkgs = . ./pkg/database/types ./pkg/encoding ./pkg/rflag ./pkg/types

install-std: test
	go build -ldflags "$(ldflagsversion) -s -w" -o $(stdbindir)/rexplorer .

install: test
	go build -race -tags "debug dev" -ldflags "$(ldflagsversion)" -o $(stdbindir)/rexplorer .

test: ineffassign unit-tests

unit-tests:
	go test -race -tags "debug testing" $(testpkgs)

ineffassign:
	ineffassign $(testpkgs)

integration-tests: integration-test-sumcoins integration-test-sumcoins-python integration-test-sumoutputs integration-test-validatevalues

integration-test-sumcoins:
	go run tests/integration/sumcoins/main.go \
		--redis-addr "$(TESTNET_REDIS_ADDR)" --redis-db "$(TESTNET_REDIS_DB)" \
		--encoding "$(TESTNET_ENCODING_TYPE)"
	go run tests/integration/sumcoins/main.go \
		--redis-addr "$(STANDARD_REDIS_ADDR)" --redis-db "$(STANDARD_REDIS_DB)" \
		--encoding "$(STANDARD_ENCODING_TYPE)"

integration-test-sumcoins-python:
	python3 tests/integration/sumcoins/main.py \
		--redis-port "$(TESTNET_REDIS_PORT)" --redis-db "$(TESTNET_REDIS_DB)" \
		--encoding "$(TESTNET_ENCODING_TYPE)"
	python3 tests/integration/sumcoins/main.py \
		--redis-port "$(STANDARD_REDIS_PORT)" --redis-db "$(STANDARD_REDIS_DB)" \
		--encoding "$(STANDARD_ENCODING_TYPE)"

integration-test-sumoutputs:
	go run tests/integration/sumoutputs/main.go \
		--redis-addr "$(TESTNET_REDIS_ADDR)" --redis-db "$(TESTNET_REDIS_DB)" \
		--encoding "$(TESTNET_ENCODING_TYPE)"
	go run tests/integration/sumoutputs/main.go \
		--redis-addr "$(STANDARD_REDIS_ADDR)" --redis-db "$(STANDARD_REDIS_DB)" \
		--encoding "$(STANDARD_ENCODING_TYPE)"

integration-test-validatevalues:
	go run tests/integration/validatevalues/main.go \
		--redis-addr "$(TESTNET_REDIS_ADDR)" --redis-db "$(TESTNET_REDIS_DB)" \
		--encoding "$(TESTNET_ENCODING_TYPE)"
	go run tests/integration/validatevalues/main.go \
		--redis-addr "$(STANDARD_REDIS_ADDR)" --redis-db "$(STANDARD_REDIS_DB)" \
		--encoding "$(STANDARD_ENCODING_TYPE)"

generate-types:
	go generate pkg/types/types.go
	go generate pkg/database/types/types.go
	go generate tests/integration/sumcoins/rtypes/generate.go
