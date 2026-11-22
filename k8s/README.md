# Локальный Inventory в kind

Манифесты описывают отдельный пример запуска Inventory и PostgreSQL в kind.
Полная система запускается через [Docker Compose](../docs/local-development.md).

Нужны kind, kubectl, Docker, Python 3 и Goose CLI. Команды выполняются из корня
проекта для выбранного локального Kubernetes context.

```bash
docker buildx build --load --build-arg SERVICE=inventory-server -t grpc-inventory:v1 .
kind load docker-image grpc-inventory:v1
```

Перед применением манифестов нужен Secret `inventory-db` с ключами
`POSTGRES_PASSWORD` и `INVENTORY_DATABASE_URL`. DSN указывает на
`inventory-postgres:5432`, базу `inventory_db` и роль `inventory`; пароль
совпадает с `POSTGRES_PASSWORD`. Значения создаются локально и не публикуются.
Для существующего PVC пароль должен соответствовать роли в PostgreSQL:
изменение Secret не меняет пароль роли.

```bash
kubectl apply -f k8s/inventory-postgres.yaml
kubectl rollout status statefulset/inventory-postgres
# В отдельном терминале:
kubectl port-forward svc/inventory-postgres 15434:5432
```

С установленным Goose и активным port-forward:

```bash
python3 scripts/migrate-inventory-kind.py
kubectl apply -f k8s/inventory-deployment.yaml -f k8s/inventory-service.yaml
kubectl rollout status deployment/inventory
kubectl port-forward svc/inventory 15053:50053
```

Скрипт читает DSN из Secret без вывода значения, заменяет адрес на локальный
port-forward и применяет существующие миграции Inventory. Локальный порт БД
можно изменить через `INVENTORY_LOCAL_DB_PORT`.

Для gRPC-клиента используется `localhost:15053` без TLS и контракт
`proto/inventory.proto`; reflection не включён. `GetInventoryByID` для
отсутствующего товара возвращает `NotFound`. Пример сообщения `CreateInventory`:

```json
{"product_id":"kind-demo-1","product_name":"Demo","category_id":"demo","stock":"10","unit_price_minor":"100"}
```

Затем тот же товар читается через `GetInventoryByID`. Повторное создание
возвращает `AlreadyExists`. Эти запросы изменяют данные локальной среды.
