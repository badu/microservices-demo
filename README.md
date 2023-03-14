# Demo Microservices

Project implements ingredients for your startup named <Insert_Element_Here>BNB, but beware that Air is taken already!

This repository represents a showcase of microservices written in Go language.

The repository was created in April 2021 (using Go 1.16) and updated in March 2023 (using Go 1.20).

Yes, it's highly over-engineered with the purpose of showcasing the "how to" different techniques and conventions.

## Setup

I'm using Linux. For other operating systems, you have to figure out the potential problems you might have.

### Prerequisites

`docker` and `docker-compose` - no instructions here.

`protoc` - I've used version 3.12.3 (you can check your version with `apt list -a protobuf-compiler`) - (to install,
run `sudo apt install -y protobuf-compiler`)

`swag` - to install, run `go install github.com/swaggo/swag/cmd/swag@v1.8.10`

`migrate` - to install run `go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@v4.15.2`

Note : you can install `swag` and `migrate` if needed by running `make install_swag_and_migrate` (NOT inside this repo)

### Steps

#### DockerCompose

* run dependencies locally `docker-compose -f docker-compose.local.yml up`

* (later) to stop containers gracefully use (so you can start them) `docker-compose -f docker-compose.local.yml stop`
* (later) ... then pick up where you stopped the containers `docker-compose -f docker-compose.local.yml start`
* (later) to delete containers `docker-compose -f docker-compose.local.yml down`

#### Containers

The following containers will get started :
`postgis_images_container` - this is the postgresql database for images microservice      
`postgis_comments_container` - this is the postgresql database for comments microservice    
`postgis_users_container` - this is the postgresql database for users microservice       
`postgis_hotels_container` - this is the postgresql database for hotels microservice

`redis_users` - this is the redis container, <TBD>

`node_exporter_container` -

`microservices-demo_rabbitmq_1` - this is the rabbitmq container, navigate
to [http://localhost:15672](http://localhost:15672) for the UI. Use `guest` for both user and password.
`grafana_container` - grafana container, navigate to [http://localhost:3000](http://localhost:3000) for the beautiful
UI. Use `admin` for both password and user.         
`prometheus_container` - prometheus container, navigate to [http://localhost:9090](http://localhost:9090) for the
UI.         
`jaeger_container` - jaeger container, navigate to [http://localhost:16686](http://localhost:16686) to see the UI.
`minio_aws_container` -

TODO : add `adminer`

## Initial Database Setup using Make

* migrate all databases `make init_databases`

## Generate Swagger

run: `make swagger_api`

### Swagger UI:

* https://localhost:8081/swagger/index.html (note this is a HTTPS route - see certificate troubleshooting)

Note : navigate to `chrome://flags/#allow-insecure-localhost` and set it to `enabled` to get rid of the warning.

## Packages choices

1. Uber Zap for logging, because I consider it to be flexible and well written. By flexible, I understand that it can
   log to Kafka if you want to.
2. Echo

## Interfaces (in Go) Design Principles:

1. Define the interface where it is being used
2. Expand request fields to make Service / Repository interface methods more readable and independent of transport layer
3. GRPC client should be passed as factory functions (which returns the actual interface of the client)
4. 
