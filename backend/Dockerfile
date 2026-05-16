FROM golang:1.22-alpine AS build
WORKDIR /src
COPY go.mod go.sum* ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o /out/server ./cmd/server && CGO_ENABLED=0 go build -o /out/worker ./cmd/worker
FROM alpine:3.20
RUN adduser -D appuser
WORKDIR /app
COPY --from=build /out/server /app/server
COPY --from=build /out/worker /app/worker
USER appuser
EXPOSE 8080
CMD ["/app/server"]
