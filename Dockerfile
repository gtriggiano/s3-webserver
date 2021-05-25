FROM golang:1.16.3 AS build
WORKDIR /go/src/github.com/gtriggiano/s3-webserver/

COPY . .
RUN go mod download
RUN go mod verify
RUN CGO_ENABLED=0 go build -o /s3-webserver ./cmd/s3-webserver.go

FROM debian:buster-slim
RUN apt-get update && apt-get install -y ca-certificates
COPY --from=build /s3-webserver /bin/s3-webserver
CMD [ "s3-webserver" ]