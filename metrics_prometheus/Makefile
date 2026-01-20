export

OTEL_SERVICE_NAME="my-example-service"
OTEL_RESOURCE_ATTRIBUTES="service.instance.id=$(shell uuidgen)"

run:
	go run main.go

metrics:
	curl http://localhost:8080
	curl http://localhost:8080/metrics

up:
	docker compose up -d


down:
	docker compose down

prom:
	open http://localhost:9090
