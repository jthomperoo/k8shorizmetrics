test:
	@echo "=============Running unit tests============="
	go test ./... -cover -coverprofile unit_cover.out

lint:
	@echo "=============Linting============="
	staticcheck ./...

format:
	@echo "=============Formatting============="
	gofmt -s -w .
	go mod tidy
	cd examples/cpureplicaprint && go mod tidy && gofmt -s -w .
	cd examples/cpuprint && go mod tidy && gofmt -s -w .

view_coverage:
	@echo "=============Loading coverage HTML============="
	go tool cover -html=unit_cover.out
