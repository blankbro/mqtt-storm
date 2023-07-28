# 构建 mac 可执行文件
go build -o output/mac/mqtt-storm cmd/mqttstorm/main.go


# 构建 linux 可执行文件
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o output/linux/mqtt-storm cmd/mqttstorm/main.go