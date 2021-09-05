CWD = $(shell pwd)
VER = $(shell git describe --tags --always --dirty)

lint:
	@-docker run --rm -t -w $(CWD) -v $(CURDIR):$(CWD) -e GOFLAGS=-mod=vendor \
		golangci/golangci-lint:v1.41.1 golangci-lint run -v
build:
	@docker build -f ./Dockerfile --build-arg VER=$(VER) -t github.com/ad/nsqauth:latest .

run: build
	@docker run --rm -e FILE='/go/bin/demoauth.csv' --mount type=bind,source=$(CWD)/demoauth.csv,target=/go/bin/demoauth.csv --name nsqauth -t github.com/ad/nsqauth:latest

build-dev:
	@docker build -f ./Dockerfile-dev --build-arg VER=$(VER) -t github.com/ad/nsqauth:dev .

dev: build-dev
	@-docker stop nsqauth-dev
	@-docker rm nsqauth-dev
	@docker run -e FILE='/go/bin/demoauth.csv' -v $(CURDIR):/app --mount type=bind,source=$(CWD)/demoauth.csv,target=/go/bin/demoauth.csv -p 7755:7755 --name nsqauth-dev -t github.com/ad/nsqauth:dev