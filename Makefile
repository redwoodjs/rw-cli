build:
	cd cli && go build -o ../rw -ldflags="-s -w -X main.version=_local_ -X main.commit=_local_ -X main.date=$(shell date '+%Y-%m-%d-%H:%M:%S')" .