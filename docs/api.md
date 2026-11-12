# REST API

[Обзор проекта](../README.md) · [Локальный запуск](local-development.md)

Контракт и ограничения ответов: [OpenAPI](../api/openapi.yaml).

| Метод | Путь | Успешный ответ |
| --- | --- | --- |
| GET | `/health` | 200, текст `OK` |
| POST | `/api/v1/inventory` | 201, созданный товар |
| GET | `/api/v1/inventory/{id}` | 200, товар и остаток |
| POST | `/api/v1/orders` | 201, `{"id":"..."}` |

Следующий пример предназначен только для отдельного тестового окружения.
Он создаёт новые тестовые записи, резервирует две единицы и оставляет их
для проверки; автоматической очистки или отмены резерва нет.
Команды выполняются в одном терминале с общими тестовыми идентификаторами:

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
504 — timeout, 500 — прочая ошибка. GET товара возвращает название
в поле `product_name`.
