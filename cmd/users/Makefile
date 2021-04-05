.PHONY:

# ==============================================================================
# Generate swagger documentation
# ==============================================================================
swagger_users_api:
	echo "Starting swagger generating"
	swag init -g **/**/*.go

# ==============================================================================
# Go migrate postgresql user service
# ==============================================================================

user_dbname = users
user_port = 5433
user_SSL_MODE = disable

force_users_db:
	migrate -database postgres://postgres:postgres@localhost:$(user_port)/$(user_dbname)?sslmode=$(user_SSL_MODE) -path migrations force 1

version_users_db:
	migrate -database postgres://postgres:postgres@localhost:$(user_port)/$(user_dbname)?sslmode=$(user_SSL_MODE) -path migrations version

migrate_users_db_up:
	migrate -database postgres://postgres:postgres@localhost:$(user_port)/$(user_dbname)?sslmode=$(user_SSL_MODE) -path migrations up 1

migrate_users_db_down:
	migrate -database postgres://postgres:postgres@localhost:$(user_port)/$(user_dbname)?sslmode=$(user_SSL_MODE) -path migrations down 1



# ==============================================================================
# Make local SSL Certificate
# ==============================================================================

make_cert:
	echo "Generating SSL certificates"
	sh ./user/ssl/instructions.sh

