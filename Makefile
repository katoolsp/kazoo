all:
	go build -o kazoo kazoo.go

linux:
	GOOS=linux GOARCH=amd64 go build -o kazoo-linux-amd64 kazoo.go

macos:
	GOOS=darwin GOARCH=amd64 go build -o kazoo-darwin-amd64 kazoo.go

windows:
	GOOS=windows GOARCH=amd64 go build -o kazoo-windows-amd64.exe kazoo.go

clean:
	rm -f kazoo kazoo-linux-amd64 kazoo-darwin-amd64 kazoo-windows-amd64.exe

install:
	mv kazoo /usr/local/bin/
	chmod +x /usr/local/bin/
