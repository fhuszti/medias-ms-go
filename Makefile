.PHONY: test slow-test full-test

migration:
	bash scripts/create_migration.sh

test:
	go clean -testcache
	go test ./...

slow-test:
	go clean -testcache
	cd test/ && go test ./...

full-test:
	make test
	make slow-test

clean:
	golangci-lint run