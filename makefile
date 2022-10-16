.SILENT:

clidt:
	go run cmd/clientDataTransfer/main.go

servdt:
	go run cmd/serverDataTransfer/main.go

test:
	go test ./... -v