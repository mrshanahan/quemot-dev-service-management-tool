.DEFAULT: build

build:
	go build -o ./cmd/smt ./cmd/smt.go

install:
	go build -o ~/bin/smt ./cmd/smt.go

.PHONY: build install