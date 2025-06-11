.PHONY: test unit-tests functional-tests

migration:
	bash scripts/create_migration.sh

unit-tests:
	go clean -testcache
	go test ./...

e2e-tests:
	go clean -testcache
	cd test/e2e/ && go test ./...

integration-tests:
	go clean -testcache
	cd test/integration/ && go test ./...

functional-tests:
	go clean -testcache
	cd test/ && go test ./...

test:
	make unit-tests
	make functional-tests

clean:
	golangci-lint run

optimise-backlog:
	go run .\cmd\optimise-backlog\
