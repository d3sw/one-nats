go test -coverprofile test.out %1
go tool cover -html test.out