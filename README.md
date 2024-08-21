# Описание прокта

Сервис для хранения данных. Позволяет загружать и выгружать данные с любым из известных mime-типов, в том числе различные файлы большого размера (с учетом физических ограничений сервера, хранилища и конфигурации оных).

# Как запустить

## В контейнере
1. `git clone https://github.com/ArtemVoronov/clearway-task-assets-service.git`
2. `run.sh start`

## На другой среде
1. `git clone https://github.com/ArtemVoronov/clearway-task-assets-service.git`
2. настроить среду по образу и подобию `docker-compose.yaml` (PostgreSQL с шардированными и нешардированные базами, запустить скрипты миграции liquibase)
3. создать конфиг файл `.env` (см. секцию с описанием конфигурации)
4. сгенерировать сертификаты (например, через `bash`-скрипт в корне проекта: `run.sh certs`) или получить их иными удобным способом и положить в корень проекта
4. `go run .`

# Конфигурация

По-умолчанию приложение читает файл `.env`. При помощи системной переменной `CONFIG_FILE_PATH` можно указать путь до другого файла. При запуске в контейнере `CONFIG_FILE_PATH` равен `.env.dev` (см. Dokerfile в корне проекта). При обычном запуске нужно в корне проекта создать `.env` по образу и подобию примера в корне проекте (см. файл `.env.dev`):
```
# common app settings
APP_REST_API_PORT=3005
APP_TLS_CERT_PATH=server.crt
APP_TLS_KEY_PATH=server.key
APP_SERVER_READ_TIMEOUT=15m
APP_SERVER_WRITE_TIMEOUT=15m
APP_SERVER_GRACEFUL_SHUTDOWN_TIMEOUT=2m
APP_ENABLE_RUNTIME_MONITORING=false

# db settings
DATABASE_HOST=postgres
DATABASE_PORT=5432
DATABASE_USER=assets_service_user
DATABASE_PASSWORD=assets_service_password
DATABASE_NAME_PREFIX=assets_service_db
DATABASE_SHARDS_COUNT=2

DATABASE_QUERY_TIMEOUT=15m
DATABASE_CONNECT_TIMEOUT_IN_SECONDS=60
DATABASE_CONNECTIONS_POOL_MAX_CONN_LIFE_TIME=1h
DATABASE_CONNECTIONS_POOL_MAX_CONN_IDLE_TIME=30m
DATABASE_CONNECTIONS_POOL_MAX_CONNS=4
DATABASE_CONNECTIONS_POOL_MIN_CONNS=0
DATABASE_CONNECTIONS_POOL_HEALTH_CHECK_PERIOD=1m

# for liquibase
DATABASE_URL=jdbc:postgresql://postgres:5432/assets_service_db

# http server settings
# 12 Gb
HTTP_REQUEST_BODY_MAX_SIZE_IN_BYTES=12884901888

# auth
AUTH_ACCESS_TOKEN_TTL=24h

# cors
CORS_ALLOWED_ORIGIN=*
CORS_ALLOWED_HEADERS=X-Requested-With
CORS_ALLOWED_METHODS=GET,POST,PUT,DELETE,OPTIONS
```

# REST API

- `GET /api/doc/` - HTML-страница Swagger UI со спецификацией сервиса в формате OpenAPI Specification 2.0
- `GET /api/` - спецификация сервиса в формате OpenAPI Specification 2.0
- `GET /health` - кумулятивная информация о готовности и работоспособности сервиса
- `POST /api/users` - создать пользователя, заголовок авторизации не требуется
- `POST /api/auth` - аутентификация пользователя, заголовок авторизации не требуется
- `GET /api/assets` - получить список данных пользователя, требуется заголовок авторизации
- `POST /api/upload-asset/{name}` - загрузить данные в сервис, требуется заголовок авторизации
- `GET /api/asset/{name}` - получить данные из сервисы, требуется заголовок авторизации
- `DELETE /api/asset/{name}` - удалить данные, требуется заголовок авторизации

# Дополнительные комментарии

TODO