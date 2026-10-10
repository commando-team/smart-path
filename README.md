# smart-path

## Запуск

```bash
docker compose -f 'compose.dev.yaml' up -d --build

go run ./cmd/api
```

## Swagger

http://localhost:8080/swagger/index.html

## Требования для разработки

- sqlc
- swag
- mockery

### Установка требований

```bash
go install github.com/sqlc-dev/sqlc/cmd/sqlc@latest

go install github.com/swaggo/swag/cmd/swag@latest

go install github.com/vektra/mockery/v3@v3.8.0
```

## Генерация документации API

```bash
swag init -o ./docs -g ./cmd/api/main.go --parseInternal
```

## Генерация кода для работы с БД

```bash
sqlc generate
```