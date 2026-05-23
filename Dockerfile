FROM golang:1.26-alpine AS build
WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 go build -trimpath -o /bin/server ./cmd/server

FROM alpine:3.21
RUN adduser -D -u 10001 app
COPY --from=build /bin/server /bin/server
USER app
EXPOSE 8080
ENTRYPOINT ["/bin/server"]
