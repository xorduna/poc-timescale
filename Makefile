# Variables
CONTAINER_NAME = timescaledb
DB_PASSWORD = mysecretpassword
DB_PORT = 5432
VOLUME_NAME = timescale_data

# Commands
.PHONY: start stop restart rm logs ps

# Database Commands

db-start:
	docker run -d --name $(CONTAINER_NAME) \
		-p $(DB_PORT):5432 \
		-e POSTGRES_PASSWORD=$(DB_PASSWORD) \
		-v $(VOLUME_NAME):/var/lib/postgresql/data \
		timescale/timescaledb:latest-pg16

db-stop:
	docker stop $(CONTAINER_NAME)

db-restart: stop start

db-rm: stop
	docker rm $(CONTAINER_NAME)

db-logs:
	docker logs $(CONTAINER_NAME)

db-ps:
	docker ps -a | grep $(CONTAINER_NAME)

# Connect to database
db-connect:
	docker exec -it $(CONTAINER_NAME) psql -U postgres

# Run Model
run-sql:
	docker exec -i $(CONTAINER_NAME) psql -U postgres < model.sql


# Client and server
build: client/main.go server/main.go
	go build -o pts-server server/main.go
	go build -o pts-client client/main.go

