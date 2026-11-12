# grpc-course

Учебный MVP на Go: HTTP API для товаров и заказов поверх трёх gRPC-сервисов.
Order резервирует товар в Inventory, сохраняет заказ и вызывает Payment.
Payment сохраняет запись о платеже; интеграции с платёжным провайдером нет.
Первая версия: `v0.1.0`.

## Архитектура и стек

| Компонент | Назначение | Адрес по умолчанию |
| --- | --- | --- |
| API Gateway | REST/JSON → gRPC Order и Inventory | HTTP `:8080` |
| Order | Оркестрация резервирования, заказа и платежа | gRPC `localhost:50051` |
| Inventory | Товары, остатки, резервирование | gRPC `localhost:50053` |
| Payment | Хранение платежей с уникальным order_id | gRPC с mTLS `localhost:50052` |

У каждого сервиса своя PostgreSQL-база. Соединение Order → Payment использует
mTLS; Gateway → Order/Inventory и Order → Inventory используют plaintext gRPC.
Подробнее: [архитектура и последовательности](docs/architecture.md).

Стек: Go 1.27, gRPC/Protocol Buffers, PostgreSQL, pgx через `database/sql`,
Squirrel, Goose, Prometheus, slog, retry middleware и gobreaker для Payment.
Тесты используют стандартный `testing`, Testify и Testcontainers.

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
точки входа, для MVP запускайте четыре приложения из `cmd/` выше.

Goose можно установить в отдельный каталог без изменения зависимостей проекта:

```bash
go install github.com/pressly/goose/v3/cmd/goose@v3.28.0
```

## PostgreSQL и обязательная конфигурация

Подготовьте отдельный локальный PostgreSQL-кластер. Например, в интерактивном
`psql`, подключённом администратором к базе `postgres`, создайте роли и базы:

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

`\password` запрашивает собственный пароль интерактивно. Не повторяйте команды
создания для уже существующих баз. Для примера ниже настройте пользовательский
`~/.pgpass` с правами `0600`; пароли в репозиторий не помещаются. Можно вместо
этого получить полный DSN из вашего хранилища секретов в переменные окружения.

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

Примените миграции к своим базам из корня проекта:

```bash
goose -dir migrations/orders postgres "$ORDER_DATABASE_URL" up
goose -dir migrations/inventory postgres "$INVENTORY_DATABASE_URL" up
goose -dir migrations/payments postgres "$PAYMENT_DATABASE_URL" up
```

При передаче DSN в командной строке Goose он может быть виден в списке процессов.
Для DSN с паролем используйте поддерживаемые Goose переменные `GOOSE_DRIVER`
и `GOOSE_DBSTRING` вместо аргумента DSN. Пример для Order:

```bash
GOOSE_DRIVER=postgres GOOSE_DBSTRING="$ORDER_DATABASE_URL" goose -dir migrations/orders up
```

## Локальные mTLS-сертификаты

Order читает `certs/ca.crt`, `certs/order-client.crt`, `certs/order-client.key`;
Payment — `certs/ca.crt`, `certs/payment-server.crt`, `certs/payment-server.key`.
Payment проверяет клиентский сертификат, Order проверяет серверный с
`ServerName=localhost`. Приватный ключ CA приложениям не нужен.

Для нового клона следующий Bash-блок создаёт собственный локальный CA и
сертификаты на 30 дней. Он остановится, если `certs/` уже существует, чтобы
не перезаписать ваши ключи. Существующий CA/ключи без согласования не заменяйте.

```bash
(
  set -eu
  umask 077
  if [ -e certs ]; then
    echo 'certs/ already exists; use your existing certificates or a fresh clone' >&2
    exit 1
  fi
  mkdir certs
  openssl req -x509 -newkey rsa:3072 -nodes -sha256 -days 30 \
    -keyout certs/ca.key -out certs/ca.crt -subj '/CN=grpc-course-local-ca' \
    -addext 'basicConstraints=critical,CA:TRUE' \
    -addext 'keyUsage=critical,keyCertSign,cRLSign'
  openssl req -newkey rsa:3072 -nodes -sha256 \
    -keyout certs/payment-server.key -out certs/payment-server.csr \
    -subj '/CN=localhost'
  printf '%s\n' 'subjectAltName=DNS:localhost,IP:127.0.0.1' \
    'extendedKeyUsage=serverAuth' 'keyUsage=digitalSignature,keyEncipherment' \
    > certs/payment-server.ext
  openssl x509 -req -sha256 -days 30 -in certs/payment-server.csr \
    -CA certs/ca.crt -CAkey certs/ca.key -CAcreateserial \
    -out certs/payment-server.crt -extfile certs/payment-server.ext
  openssl req -newkey rsa:3072 -nodes -sha256 \
    -keyout certs/order-client.key -out certs/order-client.csr \
    -subj '/CN=grpc-course-order-client'
  printf '%s\n' 'extendedKeyUsage=clientAuth' 'keyUsage=digitalSignature' \
    > certs/order-client.ext
  openssl x509 -req -sha256 -days 30 -in certs/order-client.csr \
    -CA certs/ca.crt -CAkey certs/ca.key -CAserial certs/ca.srl \
    -out certs/order-client.crt -extfile certs/order-client.ext
  openssl verify -CAfile certs/ca.crt -purpose sslserver certs/payment-server.crt
  openssl verify -CAfile certs/ca.crt -purpose sslclient certs/order-client.crt
)
```

`certs/`, приватные ключи, локальные сертификаты и `.env` исключены из Git и
Docker build context. Это локальные тестовые сертификаты, не комплект для публикации.

## Запуск четырёх приложений

После настройки DSN, миграций и сертификатов выполните **из корня проекта**
в четырёх отдельных терминалах (переменные окружения должны быть доступны):

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

`ADDRESS` переопределяет адрес конкретного gRPC-сервиса; задавайте его отдельно
для каждого процесса. У Order доступны `PAYMENT_SERVICE_ADDR` (по умолчанию
`localhost:50052`) и `INVENTORY_SERVICE_ADDRESS` (`localhost:50053`). Gateway
использует фиксированные локальные адреса Order и Inventory и слушает все
интерфейсы на `:8080`; для этого MVP ограничьте доступ локальной машиной.

```bash
curl -i http://localhost:8080/health
```

Ожидается `200 OK` и тело `OK`. Это проверка самого Gateway; состояние БД и
доступность зависимостей она не подтверждает.

## REST API и ручной smoke test

Контракт и ограничения ответов: [api/openapi.yaml](api/openapi.yaml).

| Метод | Путь | Успешный ответ |
| --- | --- | --- |
| GET | `/health` | 200, текст `OK` |
| POST | `/api/v1/inventory` | 201, созданный товар |
| GET | `/api/v1/inventory/{id}` | 200, товар и остаток |
| POST | `/api/v1/orders` | 201, `{"id":"..."}` |

Следующий пример предназначен только для отдельного тестового окружения.
Он создаёт новые тестовые записи, резервирует две единицы и оставляет их
для проверки; автоматической очистки или отмены резерва нет.
Используйте один терминал, чтобы сохранить уникальные идентификаторы:

```bash
SMOKE_PRODUCT_ID="release-smoke-$(date +%s)-$$"
SMOKE_ORDER_ID="release-order-$(date +%s)-$$"

# 1. Создать товар: ожидается 201 и stock=10.
curl -i http://localhost:8080/api/v1/inventory \
  -H 'Content-Type: application/json' \
  -d "{\"product_id\":\"$SMOKE_PRODUCT_ID\",\"product_name\":\"Release smoke\",\"category_id\":\"release-test\",\"stock\":10,\"unit_price_minor\":1500}"

# 2. Прочитать товар: ожидается 200 и stock=10.
curl -i "http://localhost:8080/api/v1/inventory/$SMOKE_PRODUCT_ID"

# 3. Создать заказ: ожидается 201.
SMOKE_ORDER_JSON=$(printf '{"id":"%s","currency":"RUB","items":[{"product_id":"%s","quantity":2,"unit_price_minor":1500}]}' "$SMOKE_ORDER_ID" "$SMOKE_PRODUCT_ID")
curl -i http://localhost:8080/api/v1/orders \
  -H 'Content-Type: application/json' -d "$SMOKE_ORDER_JSON"

# 4. Проверить остаток: ожидается stock=8.
curl -i "http://localhost:8080/api/v1/inventory/$SMOKE_PRODUCT_ID"

# 5. Повторить тот же заказ: ожидается 409 / AlreadyExists.
curl -i http://localhost:8080/api/v1/orders \
  -H 'Content-Type: application/json' -d "$SMOKE_ORDER_JSON"

# При точном повторе резерв повторно не списывается: ожидается stock=8.
curl -i "http://localhost:8080/api/v1/inventory/$SMOKE_PRODUCT_ID"
```

В JSON используются числа, не строки. Неизвестные поля отклоняются.
Ошибки имеют вид `{"error":{"code":"AlreadyExists","message":"order creation failed"}}`.
Основные статусы: 400 — входные данные, 404 — товар не найден,
409 — дубликат/конфликт резерва/недостаток остатка, 503 — сервис недоступен,
504 — timeout, 500 — прочая ошибка. GET товара сейчас ошибочно возвращает
`product_id` в поле `product_name`; учитывайте это при проверке.

## Наблюдаемость

Order предоставляет `http://localhost:9091/metrics`: `orders_created_total`,
`grpc_server_requests_total`, `grpc_server_duration_seconds` и стандартные
Go/process metrics. Счётчик orders увеличивается после успешного платежа.

Prometheus-конфигурация: [monitoring/prometheus/prometheus.yml](monitoring/prometheus/prometheus.yml),
интервал 15 секунд, target `localhost:9091`. Для запуска установленного Prometheus
на той же машине/в той же сетевой среде:

```bash
prometheus --config.file=monitoring/prometheus/prometheus.yml
```

Grafana может использовать этот Prometheus как источник данных (обычно
`http://localhost:9090` при локальном запуске). Grafana provisioning и готовых
дашбордов в репозитории нет; её запуск и настройка выполняются отдельно.

## Тесты

```bash
go test -count=1 ./...
```

PostgreSQL-тесты в `internal/order/adapters/postgres` и
`internal/inventory/adapters/postgres` автоматически создают временные БД через
Testcontainers. Нужны доступный Docker daemon и образы PostgreSQL 18 Alpine и
служебных контейнеров. Пароль каждого контейнера генерируется при запуске.
Если Docker недоступен, эти пакеты завершаются с ошибкой.

Для всех остальных пакетов без инфраструктуры (Bash):

```bash
mapfile -t UNIT_PACKAGES < <(go list ./... | awk '$0 !~ /\/e2e$/ && $0 !~ /\/internal\/(order|inventory)\/adapters\/postgres$/')
go test -count=1 "${UNIT_PACKAGES[@]}"
```

E2E по умолчанию пропускается. Для явного запуска нужны `RUN_E2E=1`,
`E2E_ORDER_DATABASE_URL`, `E2E_PAYMENT_DATABASE_URL` — DSN отдельных тестовых
баз, к которым подключены сервисы. `E2E_ORDER_SERVICE_ADDR` опционален
(по умолчанию `localhost:50051`). При включённом E2E без DSN тест падает до сети.

```bash
RUN_E2E=1 go test -count=1 -v ./e2e
```

Нужны Order, Inventory, Payment, применённые миграции, mTLS и товар `product-001`
с остатком минимум 2. Тест проверяет заказ и платёж на 1000 в минимальных
единицах USD. Заказ и платёж очищаются, резерв Inventory сохраняется;
восстанавливайте отдельное тестовое окружение перед повторным запуском.

## Ограничения MVP и статус релиза

- Заказ поддерживает ровно одну позицию. Цена берётся из запроса и не сверяется
  с Inventory; валюта проверяется только на непустоту.
- Нет общей транзакции между сервисами, отмены резерва, saga/outbox и восстановления
  незавершённого платежа. Ошибка HTTP не означает отсутствие побочных эффектов.
- Резерв идемпотентен только для того же order_id, product_id и quantity.
  Повторный Order возвращает 409, а не исходный успешный результат. Другая
  позиция с тем же order_id даёт конфликт. Payment повторяет существующий
  платёж по order_id без сверки суммы/валюты. Подробнее в архитектуре.
- GET товара возвращает неверное product_name. HTTP-аутентификации и TLS на
  Gateway нет; остальные gRPC-соединения, кроме Order → Payment, без TLS.
- Нет REST-чтения заказа, HTTP-эндпоинта Payment, изменения/пополнения товара.
  Учебный WatchOrder выдаёт демонстрационные статусы, не читает состояние БД.
- Dockerfile для Order и Inventory и локальные kind-манифесты Inventory имеются,
  но полный контейнерный запуск не подтверждён. Docker Compose отсутствует.
  Развитие Docker/Kubernetes — после первого релиза.

Результаты фактических проверок и оставшиеся ограничения:
[RELEASE_REVIEW.md](RELEASE_REVIEW.md). Не включайте в первый коммит или release
assets локальные сертификаты, ключи, `LOCAL_DATABASES.md` и старый бинарник
`order-service`: в нём ранее найдены учётные данные. Публикация этого артефакта
блокируется до отдельного согласования его устранения. `.gitignore` не заменяет
проверку подготовленного индекса и файлов релиза.
