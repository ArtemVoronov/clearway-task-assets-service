#!/bin/sh

down() {
    docker-compose down
}  

purge() {
    docker volume rm clearway-task-assets-service_database-volume
}  

build() {
    docker-compose build api
}

start() {
    docker-compose up -d
}

tail() {
    docker-compose logs -f
}

generateSelfSignedCerts() {
    openssl genrsa 2048 > server.key
    openssl req -new -x509 -nodes -sha256 -days 365 -key server.key -out server.crt -subj "/C=RU/ST=/L=/O=/OU=/CN=*.clearway-task-example.ru/emailAddress=voronov54@gmail.com"
}

db() {
    docker exec -it clearway-task-assets-service_postgres_1 psql -d assets_service_db_1 -U assets_service_user
}

case "$1" in
  start)
    down
    build
    start
    tail
    ;;
  stop)
    down
    ;;
  tail)
    tail
    ;;
  purge)
    down
    purge
    ;;
  certs)
    generateSelfSignedCerts
    ;;
  db)
    db
    ;;
  *)
    echo "Usage: $0 {start|stop|tail|purge|certs|db}"
esac