CWD = $(shell pwd)
VER = $(shell git describe --tags --always --dirty)

lint:
	@-docker run --rm -t -w $(CWD) -v $(CURDIR):$(CWD) -e GOFLAGS=-mod=vendor \
		golangci/golangci-lint:v1.41.1 golangci-lint run -v
build:
	@VER=$(VER) docker build -f ./Dockerfile -t github.com/ad/nsqauth:latest .

run: build
	@docker run --rm -e FILE='/go/bin/demoauth.csv' --mount type=bind,source=$(CWD)/demoauth.csv,target=/go/bin/demoauth.csv --name nsqauth -t github.com/ad/nsqauth:latest