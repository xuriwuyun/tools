# RUN
go mod tidy

go run cobraclient.go event --concurrent 10 --count 1000
