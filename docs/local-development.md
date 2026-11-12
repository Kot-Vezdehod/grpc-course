# Локальная разработка

[Обзор проекта](../README.md) · [mTLS](mtls.md) · [API](api.md) · [Тесты](testing.md)

## Требования и сборка

- Go 1.27.1 или новее в ветке Go 1.27, доступ к зависимостям или готовый module cache.
- PostgreSQL 18 для воспроизведения тестовой среды, три отдельные базы и роли.
- Goose CLI, совместимый с миграциями проекта; версия зависимости — v3.28.0.
- OpenSSL и Bash для инструкции сертификатов; curl для REST-примеров.
- Docker daemon нужен для Testcontainers, а не для обычного запуска бинарников.
  Prometheus и Grafana опциональны.

Из корня проекта:

```bash
go mod download
go build ./...
go vet ./...
mkdir -p bin
go build -o bin/order-server ./cmd/order-server
go build -o bin/inventory-server ./cmd/inventory-server
go build -o bin/payment-server ./cmd/payment-server
go build -o bin/api-gateway ./cmd/api-gateway
```

Сборка использует `cmd/`, `internal/`, `gen/`, `go.mod`, `go.sum`.
Контракты находятся в `proto/`; готовые Go-файлы в `gen/` сохранены в проекте,
поэтому обычная сборка не требует protoc. `main.go` и `cmd/client/` — учебные
точки входа. Основной сценарий использует четыре приложения из `cmd/`.

Установка Goose CLI без изменения зависимостей проекта:

```bash
go install github.com/pressly/goose/v3/cmd/goose@v3.28.0
```

## PostgreSQL и конфигурация

Для локального запуска используются три отдельные базы. Пример создания ролей
и баз в интерактивном `psql` с административным подключением к `postgres`:

```sql
CREATE USER grpc_order;
\password grpc_order
CREATE DATABASE grpc_orders OWNER grpc_order;
CREATE USER grpc_inventory;
\password grpc_inventory
CREATE DATABASE grpc_inventory OWNER grpc_inventory;
CREATE USER grpc_payment;
\password grpc_payment
CREATE DATABASE grpc_payments OWNER grpc_payment;
```

`\password` запрашивает пароль интерактивно. Команды создания предназначены
для новой базы. В примере ниже аутентификация использует пользовательский
`~/.pgpass` с правами `0600`. Альтернатива — полный DSN, загруженный в переменные
окружения из хранилища секретов.

Пример для трёх баз на одном локальном PostgreSQL, без паролей в командах:

```bash
export ORDER_DATABASE_URL='host=127.0.0.1 port=5432 dbname=grpc_orders user=grpc_order sslmode=disable'
export INVENTORY_DATABASE_URL='host=127.0.0.1 port=5432 dbname=grpc_inventory user=grpc_inventory sslmode=disable'
export PAYMENT_DATABASE_URL='host=127.0.0.1 port=5432 dbname=grpc_payments user=grpc_payment sslmode=disable'
```

| Переменная | Где обязательна |
| --- | --- |
| `ORDER_DATABASE_URL` | Order |
| `INVENTORY_DATABASE_URL` | Inventory |
| `PAYMENT_DATABASE_URL` | Payment |

Переменные экспортируются в терминале запуска соответствующего сервиса.
Fallback DSN нет: отсутствие, пустое или пробельное значение даёт ошибку
`<ИМЯ_ПЕРЕМЕННОЙ> is required` до подключения к БД. `.env` автоматически не
загружается. `sslmode=disable` в примере относится к локальной PostgreSQL;
это отдельное соединение от gRPC mTLS.

Применение миграций из корня проекта:

```bash
goose -dir migrations/orders postgres "$ORDER_DATABASE_URL" up
goose -dir migrations/inventory postgres "$INVENTORY_DATABASE_URL" up
goose -dir migrations/payments postgres "$PAYMENT_DATABASE_URL" up
```

При передаче DSN в командной строке Goose он может быть виден в списке процессов.
Для DSN с паролем доступны переменные Goose `GOOSE_DRIVER` и `GOOSE_DBSTRING`,
которые позволяют не передавать строку подключения в аргументах. Пример для Order:

```bash
GOOSE_DRIVER=postgres GOOSE_DBSTRING="$ORDER_DATABASE_URL" goose -dir migrations/orders up
```

## Сертификаты

Перед запуском Order и Payment необходимы [локальные mTLS-сертификаты](mtls.md).

## Запуск приложений

После настройки DSN, миграций и сертификатов приложения запускаются из корня
проекта в четырёх отдельных терминалах с соответствующими переменными окружения:

```bash
# Терминал 1 — Payment
./bin/payment-server
# Терминал 2 — Inventory
./bin/inventory-server
# Терминал 3 — Order
./bin/order-server
# Терминал 4 — Gateway
./bin/api-gateway
```

`ADDRESS` переопределяет адрес конкретного gRPC-сервиса и задаётся для каждого
процесса отдельно. У Order доступны `PAYMENT_SERVICE_ADDR` (по умолчанию
`localhost:50052`) и `INVENTORY_SERVICE_ADDRESS` (`localhost:50053`). Gateway
использует фиксированные локальные адреса Order и Inventory и слушает все
интерфейсы на `:8080`. Конфигурация рассчитана на локальную разработку.

```bash
curl -i http://localhost:8080/health
```

Ожидается `200 OK` и тело `OK`. Это проверка самого Gateway; состояние БД и
доступность зависимостей она не подтверждает.
