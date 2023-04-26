build:
	go mod tidy
	gofumpt -w main.go
	goreleaser build --single-target --snapshot --clean
