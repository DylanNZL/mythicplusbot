build:
	go build main.go
run:
	go run main.go

# Download and build the golangci-lint binary so we can lint the project
bin/golangci-lint:
	@go build -o $@ github.com/golangci/golangci-lint/cmd/golangci-lint

lint: bin/golangci-lint
	bin/golangci-lint run --config .golangci.yml

test:
	go test -v ./...