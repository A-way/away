
ifeq ($(shell go env GOOS), darwin)
	CGO_CFLAGS = $(shell go env CGO_CFLAGS)
	CGO_CFLAGS += -mmacosx-version-min=10.13
endif

ifneq ($(CGO_CFLAGS),)
	CGO_CFLAGS := CGO_CFLAGS="$(CGO_CFLAGS)"
endif


build:
	go build -o away -v

run: build
	./away -lp 1080 -rp 8080

lib:
	$(CGO_CFLAGS) go build -buildmode c-archive -ldflags "-s -w" -tags lib -o libaway.a -v

clean:
	go clean

win64:
	GOOS=windows GOARCH=amd64 go build -ldflags "-s -w" -o away.exe -v

.PHONY: clean run
