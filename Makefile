build:
	cd cli && go build -o ../rw -ldflags="-s -w -X main.version=_local_ -X main.commit=_local_ -X main.date=$(shell date '+%Y-%m-%d-%H:%M:%S')" .

smoke-test: build
	./rw --version

capture-telemetry:
	docker run --rm --name jaeger \
		-e COLLECTOR_ZIPKIN_HOST_PORT=:9411 \
		-p 6831:6831/udp \
		-p 6832:6832/udp \
		-p 5778:5778 \
		-p 16686:16686 \
		-p 4317:4317 \
		-p 4318:4318 \
		-p 14250:14250 \
		-p 14268:14268 \
		-p 14269:14269 \
		-p 9411:9411 \
		jaegertracing/all-in-one:1.53
