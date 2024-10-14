tidy:
	go mod tidy

lint:
	golangci-lint run

lint_fix:
	golangci-lint run --fix

build:
	go build -o reservation main.go

build-release:
	go build -ldflags="-s -w -X 'main.Version=$(version)'" -o reservation main.go

build-docker:
	docker build --build-arg VERSION=${VERSION} --build-arg RELEASE=${RELEASE} -t tateexon/reservation:latest .

test:
	go test -timeout 5m -cover -covermode=count ./...

typos:
	typos

generate:
	oapi-codegen -config ./schema/oapi-codegen-config.yaml ./schema/openapi.yaml

start-local: build-docker
	docker compose up -d

stop-local:
	docker compose down -vv
