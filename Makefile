build:
	go build ./...

run: build
	go run ./examples/mdf-reader samples/sample3.mf4

test:
	go test -v ./... -count=1 -v
