# Этап сборки
FROM golang:1.20-alpine AS builder
WORKDIR /app

# Скопируем go.mod и go.sum
COPY go.mod go.sum ./
RUN go mod download

# Копируем всё остальное и билдим
COPY . .
RUN go build -o server ./cmd/server

# Финальный образ
FROM alpine:3.17
WORKDIR /app

# Копируем бинарник и скрипты
COPY --from=builder /app/server /app/server
COPY --from=builder /app/scripts /app/scripts

# Укажем, что мы слушаем 8080
EXPOSE 8080

# Можно указать ENV переменные (дефолтные)
ENV DB_HOST=localhost
ENV DB_PORT=5432
ENV DB_USER=validator
ENV DB_PASSWORD=val1dat0r
ENV DB_NAME=project-sem-1

# Запускаем
CMD ["/app/scripts/run.sh"]
