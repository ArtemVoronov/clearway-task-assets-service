# Описание прокта

Сервис для хранения данных. Позволяет загружать и выгружать данные любого типа, в том числе файлы большого размера (с учетом физических ограничений сервера, хранилища и конфигурации оных).

# Как запустить

## В контейнере
1. `git clone https://github.com/ArtemVoronov/clearway-task-assets-service.git`
2. `cd clearway-task-assets-service`
3. `./run.sh prepare`
4. `./run.sh start`

По-умолчанию приложение будет доступно по `https://localhost:3005`. Для остановки контейнеров можно использовать `./run.sh stop`. А если нужно остановить контейнеры, удалить контейнеры и volumes, то `./run.sh purge`

## На другой среде
1. `git clone https://github.com/ArtemVoronov/clearway-task-assets-service.git`
2. `cd clearway-task-assets-service`
3. настроить среду по образу и подобию `docker-compose.yaml` (PostgreSQL с шардированными и нешардированными базами, запустить скрипты миграции `liquibase`)
4. создать конфиг файл `.env` (см. секцию с описанием конфигурации)
5. сгенерировать сертификаты (например, через `bash`-скрипт в корне проекта: `run.sh certs`) или получить их иными удобным способом и положить в корень проекта
6. `go run .`

Если базовая конфигурация не менялась, то приложение также будет доступно по `https://localhost:3005`

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

# Описание REST API

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

1. Первые три страницы ТЗ (с общим описанием задания, требуемыми сигнатурами REST API и форматом запросов/ответов, а также запрет на использование сторонник библиотек, кроме [pgx](https://github.com/jackc/pgx)) - воспринимались, как обязательные условия реализации.
2. Страница с схемой БД - воспринимались, как некая рекомендация, т.е. итоговая реализация модели данных отличается этой и учитывает содержимое страницы ТЗ с дополнительными вопросами и задачми.
3. Все дополнительные задачи были сделаны:
 - Сервис работает по протоколу HTTPS (вопрос откуда берутся сертификаты оставил за кадром, для примера я их сам генерю, поэтому в `curl` везде надо будет добавлять флажок `--insecure` и поступать аналогично для других аналогичных инструментов)
 - В сервисе есть методы для получения списка файлов пользователя и их удаления.
 - IP адрес авторизованного пользователя сохраняется в БД. Это просто дополнительное поле в таблице с токенами.
 - Максимальное время жизни токена авторизации - 24 часа. Это конфигурируемый параметр.
 - Для каждого пользователя существует всегда 1 токен авторизации, т.е. при повторной авторизации старый токен становится невалидным.
4. Ответ на вопрос "Что можно улучшить в схеме БД?" будет несколько объёмным:
4.1 Судя по модели и общему описанию из ТЗ, тут должно было бы быть три отдельных сервиса: для авторизиации, для хранения профилей пользователей и для хранения данных. Это в том случае, если мы будем придерживаться сервисного подхода в организации нашей архитектуры. И в каждом из этих сервисов использовалась бы одна из вышеописанных таблиц. Разнесение по сервисам я делать не стал дабы не переусложнять текущее задание и в основном из-за ограничения по времени на его реализацию. Однако текущая модель данных проектировалась с учетом этой возможности в будущем, как если бы мы этот сервис стали развивать далее.
4.2 В продолжение предыдущего пункта, модель данных проектировась с учетом горизонтального масштабирования нагрузки: asset'ы шардируются по ключу, ключем является UUID пользователя, и заранее выполняется прешардинг. Для примера число шард было сделано равным 2 (это конфигурируемый параметр). Соответственно у нас имеется две БД для asset'ов. Пользователи и токены авторизации не шардируются, но для упрощения этой возможности везде используется UUID пользователя для связей между сущностями, а не id с типом bigint.
4.3 Ещё одним изменением в изначальной схеме является использование типа `large objects` вместо `bytea` для хранения непосредственных данных (asset'ов). Что позволяет хранить файлы большого размера, и наряду с шардированием по ключу UUID-пользователя позволяет хранить все файлы пользователя в одной шарде. Это локализует запросы пользователей, которые у нас изолированы согласно ТЗ (нельзя обращаться к чужим файла). В случае если бы, поступила задача сделать доступ к чужим файлам, то это просто потребовало бы построение некоего индекса, где к ссылке на файла привзяывается его местоположение (шарда). Сейчас же мы не сможем получить от сервиса `403 Forbidden` из-за этой изолированности пользователей.
4.4 Выбор UUID пользователя в качестве ключа шардирования имеет и недостатки: у нас могут быть пользователи у которых много файлов, и у котороых мало файлов. Таким образом будет неравномерно распределяться место в шардах. На этот случай можно было бы придумать некую политику ограничений для пользователей (в духе максим 100 файлов, максимальный размер всех файлов 15 Гб или что-нибудь похожее) и заранее посчитать сколько нам примерно шард потреуется и места на диске на нашу примерную аудиторию. Это концепция с политикой ограничений пока оставлена в коде ввиде отдельного `TODO`.
4.5 Несмотря на наличие поля UUID в таблице пользователей, мы сохраняем инкрементацию обычного ID - в целях удобства реализации пагинации в будущем.
5. Дополнительно был сделан метод в REST API для создания пользователей. Он не закрыт требованием наличия заголовка авторизации. Это сознательное упрощение, чтобы можно было "поиграться" с сервисом и посмотреть разные сценарии. В реальном сервисе у нас был бы некий процесс появления новых пользователей вместо этого.
6. Сценарий загрузки данных имеет один исключительный сценарий, который отличается от основного задекларированного в ТЗ. Когда пользователь загружает несколько файлов (т.е. мы имеем дело с mime-типом `multipart/form-data`), то название файла берется не из URL (`path`-параметр `{name}`), а из непосредственно тела запроса.
7. Для размера данных пользователя на текущий момент есть всего одно ограничение: в конфигурации бэкенда задаётся максимальный размер тела запроса. В базовой конфигурации оно равно 12 Гб. При превышении этого лимита соответствующий метод в REST API вернет ошибку. Это позволяет сделать разные бэкенды с разным уровнем ограничений. В целом же размеры файлов ограничены только максимально возможным размером таблиц в БД и, в частности, размером `large objects`, которые хранятся в системных таблицах (это зависит от версии PostgreSQL).
8. С помощью опции `APP_ENABLE_RUNTIME_MONITORING=true` их конфиге можно вывести в лог информацию об использовании памяти, чтобы посмотреть эффективно или неээфективно она расходуется при загрузке больших файлов и/или большого количества файлов

# TODO
- [ ] Добавить политики ограничений для пользователей (например, максмальное количество файлов, максимальный размер на один файл, максимальный размер всех файлов)
- [ ] Добавить интеграционные тесты (скажем, сделать отдельный `docker-compose-integration-tests.yml`, где через `liquibase` будут пересоздаваться базы данных в PosgreSQL для изоляции тестов, а по команде `./run.sh` будет подниматься тестовая среда и далее запускаться изолированные тесты по тэгу `integrations`, чтобы отделять это от юнит-тестов)
- [ ] Добавить механизм расшаривания данных, чтобы можно было предоставить доступ кому-либо (публичный или конкретным пользователям)
- [ ] Сделать пагинацию при получении списка файлов (`GET /api/assets`)
- [ ] Выделить отдельные сервисы для авторизации и хранение профилей пользователей
- [ ] Добавить в API метод `/metrics` для сбора метрик в формате Prometheus
- [ ] Добавить в API метод `/loggers` для получения информации о логгерах сервиса и управлении уровнями логирования
- [ ] Добавить в API метод `/info` для получения метаинформации о сборке приложения
- [ ] Добавить дополнительный механизм кэшрования файлов (скажем, если это относительно небольшой статичесий контект вроде картинок или простого текста)