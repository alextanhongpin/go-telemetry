export

OTEL_EXPORTER_OTLP_PROTOCOL=http/protobuf
OTEL_EXPORTER_OTLP_METRICS_ENDPOINT=http://localhost:9090/api/v1/otlp/v1/metrics
OTEL_TRACES_EXPORTER=none
OTEL_LOGS_EXPORTER=none
OTEL_METRIC_EXPORT_INTERVAL=15000  # Set to 15s, default 60s.
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
