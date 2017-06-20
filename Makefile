LINTER = golint -set_exit_status
PACKAGES := $(shell go list ./... | grep -v 'vendor')

.PHONY: test

test:
	rm -rf ./codecov
	mkdir -p ./codecov
	for pkg in $(PACKAGES); do mkdir -p ./codecov/$${pkg}/ && go test -coverprofile="./codecov/$${pkg}/profile.out" -covermode=atomic $$pkg; done;

coveralls: test
	rm -f ./coverage.txt
	echo "mode: atomic" > coverage.txt
	(find ./codecov -name 'profile.out' -print0 | xargs -0 cat | grep -v 'mode: ') >> coverage.txt

travis: test coveralls
