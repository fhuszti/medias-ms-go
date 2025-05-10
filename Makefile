.PHONY: test unit-tests functional-tests

migration:
	bash scripts/create_migration.sh

unit-tests:
	go clean -testcache
	go test ./...

functional-tests:
	go clean -testcache
	cd test/ && go test ./...

test:
	make unit-tests
	make functional-tests

clean:
	golangci-lint run