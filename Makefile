# By default, we tell `make` to remain silent. We can enable a more verbose
# output by passing `VERBOSE=1` to `make`.
VERBOSE ?= 0
ifeq ($(VERBOSE), 0)
.SILENT:
endif

MAKEFLAGS += --warn-undefined-variables

.DEFAULT_GOAL := help

# Disable implicit rules.
.SUFFIXES:

packages      = $(shell go list -mod vendor ./... | grep -v 'vendor')
coverage_file = coverage.txt

$(packages):
	mkdir -p codecov/$@
	go test -v -mod vendor -coverprofile="codecov/$@/profile.out" -covermode=atomic $@

# This target creates a coverage report for Codecov
$(coverage_file): test
	echo "mode: atomic" > $@
	(find ./codecov -name 'profile.out' -print0 | xargs -0 cat | grep -v 'mode: ') >> $@
	echo ""
	echo "Coverage report generated: $@"

test: ## run the tests
test: $(packages)
.PHONY: test

test-ci: ## run the tests and coverage in Circle CI
test-ci: clean $(coverage_file)
.PHONY: ci

clean: ## remove build/test artifacts
	rm -rf codecov $(coverage_file)
.PHONY: clean

help:
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'
.PHONY: help
