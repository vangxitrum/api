REMOTE_USER=root
STG_REMOTE_PATH=/mnt/staging_data/aioz-stream/bin
PROD_REMOTE_PATH=/mnt/aioz-stream/bin

API_BIN_NAME=api
API_CONTAINER_NAME=aioz-stream-api

STREAM_BIN_NAME=livestream
STREAM_CONTAINER_NAME=aioz-stream-live

GRPC_CONTAINER_NAME=aioz-stream-grpc
GRPC_BIN_NAME=grpc

# PROD DEPLOY CMD
deploy-prod: confirm build
	@ssh root@w3stream mv $(PROD_REMOTE_PATH)/$(API_BIN_NAME) $(PROD_REMOTE_PATH)/BackUps/$(API_BIN_NAME).bk`date +%Y%m%d%H`
	@scp bin/$(API_BIN_NAME)  root@w3stream:$(PROD_REMOTE_PATH)
	@ssh root@w3stream docker restart $(API_CONTAINER_NAME)

deploy-stream-prod: confirm-stream build-stream
	@ssh root@w3stream mv $(PROD_REMOTE_PATH)/$(STREAM_BIN_NAME) $(PROD_REMOTE_PATH)/BackUps/$(STREAM_BIN_NAME).bk`date +%Y%m%d%H`
	@scp bin/$(STREAM_BIN_NAME) root@w3stream:$(PROD_REMOTE_PATH)
	@ssh root@w3stream docker restart $(STREAM_CONTAINER_NAME)

deploy-grpc-prod: confirm build-grpc
	@ssh root@w3stream mv $(PROD_REMOTE_PATH)/$(GRPC_BIN_NAME) $(PROD_REMOTE_PATH)/BackUps/$(GRPC_BIN_NAME).bk`date +%Y%m%d%H`
	@scp bin/$(GRPC_BIN_NAME)  root@w3stream:$(PROD_REMOTE_PATH)
	@ssh root@w3stream docker restart $(GRPC_CONTAINER_NAME)

# STG DEPLOY CMD
deploy-stg: build
	@ssh root@w3stream-stg mv $(STG_REMOTE_PATH)/$(API_BIN_NAME) $(STG_REMOTE_PATH)/BackUps/$(API_BIN_NAME).bk`date +%Y%m%d%H`
	@scp bin/$(API_BIN_NAME)  root@w3stream-stg:$(STG_REMOTE_PATH)
	@ssh root@w3stream-stg docker restart $(API_CONTAINER_NAME)

deploy-stream-stg: build-stream
	@ssh root@w3stream-stg mv $(STG_REMOTE_PATH)/$(STREAM_BIN_NAME) $(STG_REMOTE_PATH)/BackUps/$(STREAM_BIN_NAME).bk`date +%Y%m%d%H`
	@scp bin/$(STREAM_BIN_NAME) root@w3stream-stg:$(STG_REMOTE_PATH)
	@ssh root@w3stream-stg docker restart $(STREAM_CONTAINER_NAME)

deploy-grpc-stg: build-grpc
	@ssh root@w3stream-stg mv $(STG_REMOTE_PATH)/$(GRPC_BIN_NAME) $(STG_REMOTE_PATH)/BackUps/$(GRPC_BIN_NAME).bk`date +%Y%m%d%H`
	@scp bin/$(GRPC_BIN_NAME)  root@w3stream-stg:$(STG_REMOTE_PATH)
	@ssh root@w3stream-stg docker restart $(GRPC_CONTAINER_NAME)

# RUN CMD
run: build
	@APP_ENV=debug ./bin/$(API_BIN_NAME)

run-grpc: build-grpc
	@APP_ENV=debug ./bin/$(GRPC_BIN_NAME)

# BUILD CMD
build-stream:
	@cd mediamtx && GOOS=linux GOARCH=amd64 go build -o ../bin/$(STREAM_BIN_NAME) .

build:
	@GOOS=linux GOARCH=amd64 go build -o bin/$(API_BIN_NAME) ./cmd/http/.

build-grpc:
	@GOOS=linux GOARCH=amd64 go build -o bin/$(GRPC_BIN_NAME) ./cmd/grpc/.

# UTIL CMD
confirm:
	@echo -n "Do you want to deploy in prod env? [y/N] " && read ans && [ $${ans:-N} = y ]

confirm-stream:
	@echo -n "Do you want to deploy in stream prod env? [y/N] " && read ans && [ $${ans:-N} = y ]

proto:
	@protoc -I=internal/proto --go_out=internal/proto \
	--go-grpc_out=internal/proto \
	internal/proto/*.proto

gen-swagger:
	@echo "Do you want to init swagger docs? (y/n)"
	@read -p "Enter your choice: " init_choice; \
    if [ $$init_choice = "y" ]; then \
        echo "Do you want to generate SDK? (y/n)"; \
        read -p "Enter your choice: " sdk_choice; \
        if [ $$sdk_choice = "y" ]; then \
            swag init --parseDependency -g cmd/http/main.go -q --tags \!auth,!user,!reports,!watermark && \
            swag fmt; \
        else \
            swag init --parseDependency -g cmd/http/main.go -q && \
            swag fmt; \
        fi \
    else \
        echo "Skipping docs"; \
    fi

watch:
	@air

upload-deploy-resource:
	@scp ./docker-compose.yml $(host):$(remote_path) # docker config
	@scp Dockerfile.* $(host):$(remote_path)
	@scp ./00_init.sql ./postgres-prod-entrypoint.sh ./internal/triggers/trigger.sql $(host):$(remote_path) # postgres config
	@scp ./pgcat.toml $(host):$(remote_path)
	@scp ./redis.conf $(host):$(remote_path) # redis config
    @scp ./nginx.conf ./nginx/redis_lookup.lua ./aioz-live.yml $(host):$(remote_path) # livestream config
    @scp ./prometheus.yml ./datasources.yaml $(host):$(remote_path) # monitor config



