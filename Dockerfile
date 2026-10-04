FROM golang:1 AS build
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download && go mod verify
COPY *.go ./
RUN CGO_ENABLED=0 go build -o app

FROM scratch
COPY --from=build /app/app /
COPY --from=build /etc/mime.types /etc/mime.types
EXPOSE 8099
ENTRYPOINT ["/app"]
