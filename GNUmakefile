default: build

build:
	GO111MODULE=on go build

test:
	GO111MODULE=on go test ./...

testacc:
		TF_ACC=1 go test -v ./... -timeout 10m
