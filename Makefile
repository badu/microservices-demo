.PHONY:

# ==============================================================================
# Start local dev environment
# ==============================================================================

develop:
	echo "Starting develop environment"
	docker-compose -f docker-compose.yml up --build

local:
	echo "Starting local environment"
	docker-compose -f docker-compose.local.yml up --build

# ==============================================================================
# Modules support
# ==============================================================================

mod-reset:
	git checkout -- go.mod
	go mod tidy
	go mod vendor

mod-tidy:
	go mod tidy
	go mod vendor

mod-upgrade:
	# go get $(go list -f '{{if not (or .Main .Indirect)}}{{.Path}}{{end}}' -m all)
	go get -u -t -d -v ./...
	go mod tidy
	go mod vendor

mod-clean-cache:
	go clean -modcache

# ==============================================================================
# Generate swagger documentation
# ==============================================================================

swagger_api:
	echo "Starting swagger generating"
	cd ./app/users && swag init -g *.go
	cd ..
	cd ./app/gateway && swag init -g **/*.go


# ==============================================================================
# Make local SSL Certificate
# ==============================================================================

make_cert:
	echo "Generating SSL certificates"
	sh ./user/ssl/instructions.sh

# ==============================================================================
# Docker support
# ==============================================================================

FILES := $(shell docker ps -aq)

down-local:
	docker stop $(FILES)
	docker rm $(FILES)

clean:
	docker system prune -f

logs-local:
	docker logs -f $(FILES)

# ==============================================================================
# Go migrate postgresql User service
# ==============================================================================

users_dbname = users_db
user_port = 5433
user_SSL_MODE = disable

force_users_db:
	migrate -database postgres://postgres:postgres@localhost:$(user_port)/$(users_dbname)?sslmode=$(user_SSL_MODE) -path cmd/users/migrations force 1

version_users_db:
	migrate -database postgres://postgres:postgres@localhost:$(user_port)/$(users_dbname)?sslmode=$(user_SSL_MODE) -path cmd/users/migrations version

migrate_users_db_up:
	migrate -database postgres://postgres:postgres@localhost:$(user_port)/$(users_dbname)?sslmode=$(user_SSL_MODE) -path cmd/users/migrations up 1

migrate_users_db_down:
	migrate -database postgres://postgres:postgres@localhost:$(user_port)/$(users_dbname)?sslmode=$(user_SSL_MODE) -path cmd/users/migrations down 1


# ==============================================================================
# Go migrate postgresql Images service
# ==============================================================================

images_dbname = images_db
images_port = 5434
images_SSL_MODE = disable

force_images_db:
	migrate -database postgres://postgres:postgres@localhost:$(images_port)/$(images_dbname)?sslmode=$(images_SSL_MODE) -path cmd/images/migrations force 1

version_images_db:
	migrate -database postgres://postgres:postgres@localhost:$(images_port)/$(images_dbname)?sslmode=$(images_SSL_MODE) -path cmd/images/migrations version

migrate_images_db_up:
	migrate -database postgres://postgres:postgres@localhost:$(images_port)/$(images_dbname)?sslmode=$(images_SSL_MODE) -path cmd/images/migrations up 1

migrate_images_db_down:
	migrate -database postgres://postgres:postgres@localhost:$(images_port)/$(images_dbname)?sslmode=$(images_SSL_MODE) -path cmd/images/migrations down 1


# ==============================================================================
# Go migrate postgresql Hotels service
# ==============================================================================

hotels_dbname = hotels_db
hotels_port = 5435
hotels_SSL_MODE = disable

force_hotels_db:
	migrate -database postgres://postgres:postgres@localhost:$(hotels_port)/$(hotels_dbname)?sslmode=$(hotels_SSL_MODE) -path cmd/hotels/migrations force 1

version_hotels_db:
	migrate -database postgres://postgres:postgres@localhost:$(hotels_port)/$(hotels_dbname)?sslmode=$(hotels_SSL_MODE) -path cmd/hotels/migrations version

migrate_hotels_db_up:
	migrate -database postgres://postgres:postgres@localhost:$(hotels_port)/$(hotels_dbname)?sslmode=$(hotels_SSL_MODE) -path cmd/hotels/migrations up 1

migrate_hotels_db_down:
	migrate -database postgres://postgres:postgres@localhost:$(hotels_port)/$(hotels_dbname)?sslmode=$(hotels_SSL_MODE) -path cmd/hotels/migrations down 1

# ==============================================================================
# Go migrate postgresql comments service
# ==============================================================================

comments_dbname =comments_db
comments_port = 5436
comments_SSL_MODE = disable

force_comments_db:
	migrate -database postgres://postgres:postgres@localhost:$(comments_port)/$(comments_dbname)?sslmode=$(comments_SSL_MODE) -path cmd/comments/migrations force 1

version_comments_db:
	migrate -database postgres://postgres:postgres@localhost:$(comments_port)/$(comments_dbname)?sslmode=$(comments_SSL_MODE) -path cmd/comments/migrations version

migrate_comments_db_up:
	migrate -database postgres://postgres:postgres@localhost:$(comments_port)/$(comments_dbname)?sslmode=$(comments_SSL_MODE) -path cmd/comments/migrations up 1

migrate_comments_db_down:
	migrate -database postgres://postgres:postgres@localhost:$(comments_port)/$(comments_dbname)?sslmode=$(comments_SSL_MODE) -path cmd/comments/migrations down 1

# ==============================================================================
# Compile proto
# ==============================================================================

compile-proto:
	protoc app/users/*.proto    --go_out=plugins=grpc:. --go_opt=paths=source_relative
	protoc app/sessions/*.proto --go_out=plugins=grpc:. --go_opt=paths=source_relative
	protoc app/images/*.proto   --go_out=plugins=grpc:. --go_opt=paths=source_relative
	protoc app/hotels/*.proto   --go_out=plugins=grpc:. --go_opt=paths=source_relative
	protoc app/comments/*.proto --go_out=plugins=grpc:. -I . --go_opt=paths=source_relative

# ==============================================================================
# Init databases
# ==============================================================================

init_databases:
	migrate -database postgres://postgres:postgres@localhost:$(user_port)/$(users_dbname)?sslmode=$(user_SSL_MODE) -path cmd/users/migrations up 1
	migrate -database postgres://postgres:postgres@localhost:$(images_port)/$(images_dbname)?sslmode=$(images_SSL_MODE) -path cmd/images/migrations up 1
	migrate -database postgres://postgres:postgres@localhost:$(hotels_port)/$(hotels_dbname)?sslmode=$(hotels_SSL_MODE) -path cmd/hotels/migrations up 1
	migrate -database postgres://postgres:postgres@localhost:$(comments_port)/$(comments_dbname)?sslmode=$(comments_SSL_MODE) -path cmd/comments/migrations up 1

# ==============================================================================
# Install Swag, Migrate
# ==============================================================================
install_swag_and_migrate:
	go install github.com/swaggo/swag/cmd/swag@v1.8.10
	go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@v4.15.2
