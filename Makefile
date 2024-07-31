PHONY: test.unit

test.unit:
	go test -mod=readonly -race -p 8 -cover -coverprofile=coverage.out ./...
	echo "#####################################"
	go tool cover -func coverage.out | grep total
