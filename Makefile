help:
	@echo 'CLI Makefile.'
	@echo 'Usage: make [TARGET] [EXTRA_ARGUMENTS]'

test:
	go clean -cache && go test -v ./tests

format:
	gofmt -w -s .

tidy:
	go mod tidy

verify:
	go mod verify
	
vendor:
	go mod vendor

clean-build:
	go clean -modcache
	
serve-pkg:
	pkgsite -open .

serve-docs:
	godoc -http=:6060

pkg-docs:
	pkgsite -http=:6061 -open 

webhook-swag-init:
	swag init -g ../../../cmd/webhook/main.go -d internal/app/webhook -o ./api/webhook --parseDependency