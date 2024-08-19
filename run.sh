#!/bin/sh

down() {
    docker-compose down
}  

purge() {
    docker volume rm clearway-task-assets-service-database-volume
    docker rmi clearway-task-assets-service-api:latest
}  

build() {
    docker-compose build api
}

# there could be an error with database initiation at first time
allowExecuteDatabaseScripts() {
    chmod 777 scripts/docker-postgresql-multiple-databases/create-multiple-postgresql-databases.sh
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

db1() {
    docker exec -it assets-service-database psql -d assets_service_db_assets_shard_1 -U assets_service_user
}

db2() {
    docker exec -it assets-service-database psql -d assets_service_db_assets_shard_2 -U assets_service_user
}

dbunsharded() {
    docker exec -it assets-service-database psql -d assets_service_db_unsharded -U assets_service_user
}

case "$1" in
  start)
    down
    allowExecuteDatabaseScripts
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
  db1)
    db1
    ;;
  db2)
    db2
    ;;
  dbunsharded)
    dbunsharded
    ;;
  *)
    echo "Usage: $0 {start|stop|tail|purge|certs|db1|db2|dbunsharded}"
esac