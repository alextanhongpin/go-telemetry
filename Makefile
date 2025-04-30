run:
	go run main.go

trace_http:
	go run trace_http.go

up:
	@docker-compose up -d

down:
	@docker-compose down


dashboard:
	open http://localhost:16686
	open http://localhost:9090/metrics
