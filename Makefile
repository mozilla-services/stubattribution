

LINTER = golint -set_exit_status
PACKAGES := $(shell go list ./... | grep -v 'vendor')

.PHONY: test travis

test:
	rm -rf ./codecov
	mkdir -p ./codecov
	for pkg in $(PACKAGES); do mkdir -p ./codecov/$${pkg}/ && go test -coverprofile="./codecov/$${pkg}/profile.out" -covermode=atomic $$pkg; done;

install_linter:
	go get -u github.com/golang/lint/golint

lint: install_linter
	$(LINTER) $(PACKAGES)

coverage: test
	rm -f ./coverage.txt
	(find ./codecov -name 'profile.out' -print0 | xargs -0 cat) > coverage.txt
