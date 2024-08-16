FROM golang:1.23

ENV CONFIG_FILE_PATH=.env.dev

EXPOSE 3005

RUN mkdir /app
WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download && go mod verify

COPY . ./

RUN ./run.sh certs

RUN go build -o ./clearway-task-assets-service

CMD [ "./clearway-task-assets-service" ]