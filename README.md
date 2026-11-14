# smart-path

## Запуск

```bash
docker compose -f 'compose.yaml' up -d --build
```

## Запуск симуляции

```bash
go run ./cmd/simulator/main.go -n 10 -c 2
```

- `-n` Количество запросов
- `-c` Количество потоков

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
swag init -o ./docs -g ./cmd/main.go --parseInternal
```

## Генерация кода для работы с БД

```bash
sqlc generate
```

## Генерация моков

```bash
mockery
```