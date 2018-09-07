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

testpkgs = . ./pkg/types ./pkg/encoding

install-std: test
	go build -ldflags "$(ldflagsversion) -s -w" -o $(stdbindir)/rexplorer .

install: test
	go build -race -tags "debug dev" -ldflags "$(ldflagsversion)" -o $(stdbindir)/rexplorer .

test: ineffassign unit-tests

unit-tests:
	go test -race -tags "debug testing" $(testpkgs)

ineffassign:
	ineffassign $(testpkgs)

integration-tests: integration-test-sumcoins integration-test-sumcoins-python

integration-test-sumcoins:
	go run tests/integration/sumcoins/main.go \
		--db-address "$(TESTNET_REDIS_ADDR)" --db-slot "$(TESTNET_REDIS_DB)" \
		--encoding "$(TESTNET_ENCODING_TYPE)"
	go run tests/integration/sumcoins/main.go \
		--db-address "$(STANDARD_REDIS_ADDR)" --db-slot "$(STANDARD_REDIS_DB)" \
		--encoding "$(STANDARD_ENCODING_TYPE)"


generate-python-proto-pkg:
	test -d tests/integration/sumcoins/build || mkdir tests/integration/sumcoins/build
	protoc -I=./pkg/types/  --python_out=tests/integration/sumcoins/build ./pkg/types/types.proto


integration-test-sumcoins-python: generate-python-proto-pkg

	python tests/integration/sumcoins/main.py \
		--db-port "$(TESTNET_REDIS_PORT)" --db-slot "$(STANDARD_REDIS_DB)" \
		--encoding protobuf

	python tests/integration/sumcoins/main.py \
		--db-port "$(TESTNET_REDIS_PORT)" --db-slot "$(TESTNET_REDIS_DB)" \
		--encoding "$(TESTNET_ENCODING_TYPE)"
	python tests/integration/sumcoins/main.py \
		--db-port "$(STANDARD_REDIS_PORT)" --db-slot "$(STANDARD_REDIS_DB)" \
		--encoding "$(STANDARD_ENCODING_TYPE)"



generate-types:
	go generate pkg/types/types.go
	go generate types.go
