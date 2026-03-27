.PHONY: build test install completions

build:
	go build -o bin/twine ./cmd/twine

test:
	go test ./...

install:
	go install ./cmd/twine

completions: build
	./bin/twine completion fish > completions/twine.fish
	./bin/twine completion bash > completions/twine.bash
	./bin/twine completion zsh  > completions/twine.zsh