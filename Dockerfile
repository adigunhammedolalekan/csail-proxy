FROM golang:alpine3.11

COPY . /app
WORKDIR /app

RUN go get ./...
ENV GOOS linux
RUN go build -o proxy cmd/cmd.go
EXPOSE 9093
ENTRYPOINT [ "./proxy" ]