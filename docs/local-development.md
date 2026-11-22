# Локальная разработка

[Обзор проекта](../README.md) · [mTLS](mtls.md) · [API](api.md) · [Тесты](testing.md)

## Docker Compose

Нужны Docker Engine, Compose v2 с поддержкой `up --wait`, Buildx, OpenSSL,
Bash и curl. Команды выполняются из корня проекта. Это локальная среда MVP:
снаружи доступен только HTTP Gateway на `127.0.0.1:8080`, gRPC и PostgreSQL
доступны внутри сети Compose. Сервисы находят зависимости по DNS-именам Compose.

### Сборка образов

Общий Dockerfile выбирает приложение через `SERVICE`. `--load` загружает образ
в локальный Docker Engine, где его использует Compose:

```bash
docker buildx build --load --build-arg SERVICE=api-gateway -t grpc-gateway:dev .
docker buildx build --load --build-arg SERVICE=order-server -t grpc-order:dev .
docker buildx build --load --build-arg SERVICE=inventory-server -t grpc-inventory:dev .
docker buildx build --load --build-arg SERVICE=payment-server -t grpc-payment:dev .
docker compose --profile migration build order-migrate inventory-migrate payment-migrate
```

Go toolchain остаётся в builder-stage, runtime содержит бинарник и CA-сертификаты.
Мигратор собирается отдельно из `Dockerfile.migrate` с версией Goose из `go.mod`.
У приложений в Compose указан только `image`: команда `up --build` сама
не собирает четыре Go-образа.

### Локальные секреты и сертификаты

Для нового клона следующий блок создаёт три случайных пароля с закрытыми правами.
Он остановится, если каталог уже существует:

```bash
(
  set -eu
  umask 077
  mkdir local-secrets
  openssl rand -hex 32 > local-secrets/order_password.txt
  openssl rand -hex 32 > local-secrets/inventory_password.txt
  openssl rand -hex 32 > local-secrets/payment_password.txt
)
```

Затем создаются [mTLS-сертификаты](mtls.md) по инструкции для нового клона.
Серверный сертификат содержит `DNS:payment` и `DNS:localhost`. Compose монтирует
сертификаты только для чтения; приватный ключ CA приложениям не передаётся.
`certs/` и `local-secrets/` исключены из Git и Docker build context.

Compose передаёт пароли файлами в `/run/secrets/`. Исходные файлы остаются
на хосте. Пароль PostgreSQL применяется при первой инициализации volume:
изменение файла не меняет пароль уже созданной роли.

### PostgreSQL и миграции

```bash
docker compose config -q
docker compose --profile migration config -q
docker compose up -d --wait order-db inventory-db payment-db
docker compose --profile migration run --rm order-migrate
docker compose --profile migration run --rm inventory-migrate
docker compose --profile migration run --rm payment-migrate
```

Каждая команда мигратора должна завершиться успешно до запуска приложений.
Миграторы одноразовые: читают пароль из secret в `PGPASSWORD` и запускают Goose
`up`. SQL-файлы из `migrations/` подключены read-only bind mounts, поэтому
исключение миграций из build context не мешает их применению.
Каждая БД использует отдельный именованный volume на `/var/lib/postgresql`.
Healthcheck `pg_isready` подтверждает готовность PostgreSQL принимать подключения,
но не наличие схемы или готовность приложений.

### Запуск, проверка и остановка

```bash
docker compose up -d payment inventory order api-gateway
docker compose ps
curl -i http://127.0.0.1:8080/health
```

Приложениям может потребоваться время на запуск. Ответ `200 OK` с телом `OK`
подтверждает доступность Gateway. Для проверки всей цепочки используется
[REST-сценарий](api.md): сначала создание товара, затем заказ и проверка остатка.

Остановка с сохранением контейнеров и последующий запуск:

```bash
docker compose stop
docker compose start
```

Удаление контейнеров и сетей с сохранением данных PostgreSQL:

```bash
docker compose down
```

Именованные volumes сохраняются. Флаг `-v` удаляет их вместе с данными
и для обычной остановки не используется.

## Запуск бинарников на хосте

### Требования и сборка

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
поэтому обычная сборка не требует protoc. `main.go` и `cmd/client/` — демонстрационные
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
URL имеет приоритет. Вместо URL можно задать полный набор `<SERVICE>_DB_HOST`,
`<SERVICE>_DB_PORT`, `<SERVICE>_DB_NAME`, `<SERVICE>_DB_USER` и
`<SERVICE>_DB_PASSWORD_FILE`, где `<SERVICE>` — `ORDER`, `INVENTORY` или `PAYMENT`.
В файле хранится пароль; завершающий перевод строки удаляется. Без URL или
полного набора настроек приложение завершится с ошибкой до подключения к БД. `.env` автоматически не
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
использует `ORDER_SERVICE_ADDRESS` и `INVENTORY_SERVICE_ADDRESS`
(по умолчанию `localhost:50051` и `localhost:50053`) и слушает все
интерфейсы на `:8080`. Конфигурация рассчитана на локальную разработку.

```bash
curl -i http://localhost:8080/health
```

Ожидается `200 OK` и тело `OK`. Это проверка самого Gateway; состояние БД и
доступность зависимостей она не подтверждает.
