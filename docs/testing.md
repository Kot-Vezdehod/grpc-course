# Тестирование

[Обзор проекта](../README.md) · [Настройка окружения](local-development.md)

## Полный набор проверок

```bash
go build ./...
go vet ./...
```

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
перед повторным запуском требуется восстановление тестового окружения.
