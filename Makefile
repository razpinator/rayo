# Makefile for functure project

.PHONY: test fmt lint build

test:
	go test ./...

fmt:
	go fmt ./...

lint:
	golint ./...

build:
	go build ./...

test-golden:
	go test -run TestGolden ./internal/testutil
