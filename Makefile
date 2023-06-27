build:
	go build -o bin/gomdf

run: build
	./bin/gomdf

test:
	go test -v ./... -count=1