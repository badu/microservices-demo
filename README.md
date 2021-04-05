# Demo Microservices

## Initial database setup

Run the following commands

`docker exec -it postgresql_comments bash`
`psql --host=localhost --username=postgres --password`
Type password `postgres`
`CREATE DATABASE comments;`
Type `\l` to see it created.

`docker exec -it postgresql_hotels bash`
`psql --host=localhost --username=postgres --password`
Type password `postgres`
`CREATE DATABASE hotels;`
Type `\l` to see it created.

`docker exec -it postgresql_images bash`
`psql --host=localhost --username=postgres --password`
Type password `postgres`
`CREATE DATABASE images;`
Type `\l` to see it created.

`docker exec -it postgresql_users bash`
`psql --host=localhost --username=postgres --password`
Type password `postgres`
`CREATE DATABASE users;`
Type `\l` to see it created.

## Migrate

Install [this](https://github.com/golang-migrate/migrate) and run make commands for each of the database to create tables.

## Generate Swagger

Install [this](https://github.com/swaggo/swag) then run: `make swagger_api`

### Jaeger UI:

http://localhost:16686

### Prometheus UI:

http://localhost:9090

### Grafana UI:

http://localhost:3000 user > admin password > admin

### RabbitMQ UI:

http://localhost:15672 user > guest password > guest

### Swagger UI:

* https://localhost:8081/swagger/index.html
