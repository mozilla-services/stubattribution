LINTER = golint -set_exit_status
PACKAGES := $(shell go list ./... | grep -v 'vendor')

.PHONY: test coveralls travis clean

codecov: clean
	mkdir -p codecov

test: $(PACKAGES)

$(PACKAGES): codecov
	mkdir -p codecov/$@
	go test -coverprofile="codecov/$@/profile.out" -covermode=atomic $@

coveralls: test
	echo "mode: atomic" > coverage.txt
	(find ./codecov -name 'profile.out' -print0 | xargs -0 cat | grep -v 'mode: ') >> coverage.txt

travis: test coveralls

clean:
	rm -rf codecov
	rm -f coverage.txt
