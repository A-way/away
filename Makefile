
build:
	go build -o away -v

run: build
	./away -lp 1080 -rp 8080

lib:
	go build -buildmode c-archive -ldflags "-s -w" -tags lib -o libaway.a -v

clean:
	go clean

win64:
	GOOS=windows GOARCH=amd64 go build -ldflags "-s -w" -o away.exe -v

.PHONY: clean run
