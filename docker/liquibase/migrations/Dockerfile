FROM liquibase/liquibase

ADD ./migrations /liquibase/changelog

CMD ["sh", "-c", "docker-entrypoint.sh --url=${DATABASE_URL} --username=${DATABASE_USER} --password=${DATABASE_PASSWORD} --classpath=/liquibase/changelog --changeLogFile=changelog/db.changelog-root.xml update"]