generate:
	cd ent && go run -mod=mod entc.go
	oapi-codegen --package=siteapi --generate=echo-server,strict-server,models,embedded-spec -o gen/site/site.gen.go ../openapi/site.openapi.json
	oapi-codegen --package=externalapi --generate=echo-server,strict-server,models,embedded-spec -o gen/external/external.gen.go ../openapi/external.openapi.json
	oapi-codegen --package=collectapi --generate=echo-server,strict-server,models,embedded-spec -o gen/collect/collect.gen.go ../openapi/collect.openapi.json

build:
	go build ./...

run:
	go run ./cmd/server

test:
	go test ./...

.PHONY: generate build run test
