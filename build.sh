GOOS=darwin     GOARCH=amd64    go build -ldflags '-s' -o bin/zaptun-darwin-amd64       cmd/zaptun-client/main.go
GOOS=darwin     GOARCH=arm64    go build -ldflags '-s' -o bin/zaptun-darwin-arm64       cmd/zaptun-client/main.go
GOOS=linux      GOARCH=386      go build -ldflags '-s' -o bin/zaptun-linux-386          cmd/zaptun-client/main.go
GOOS=linux      GOARCH=amd64    go build -ldflags '-s' -o bin/zaptun-linux-amd64        cmd/zaptun-client/main.go
GOOS=linux      GOARCH=arm      go build -ldflags '-s' -o bin/zaptun-linux-arm          cmd/zaptun-client/main.go
GOOS=linux      GOARCH=arm64    go build -ldflags '-s' -o bin/zaptun-linux-arm64        cmd/zaptun-client/main.go
GOOS=windows    GOARCH=386      go build -ldflags '-s' -o bin/zaptun-windows-386.exe    cmd/zaptun-client/main.go
GOOS=windows    GOARCH=amd64    go build -ldflags '-s' -o bin/zaptun-windows-amd64.exe  cmd/zaptun-client/main.go