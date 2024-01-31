build:
	cd cli && go build -o ../rw -ldflags="-s -w" .
