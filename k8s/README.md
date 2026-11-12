# Локальный Inventory в kind

Применять только Inventory; Order и Payment не затрагиваются.
Образы: `grpc-inventory:v1` (уже загружен в kind), `postgres:18`.

Перед первым apply создайте Secret `inventory-db` с ключами
`POSTGRES_PASSWORD` и `INVENTORY_DATABASE_URL`.
`INVENTORY_DATABASE_URL` должен содержать ваш DSN для хоста
`inventory-postgres:5432` и базы `inventory_db`.
Не коммитьте Secret или реальные значения. Для существующего PVC пароль должен
соответствовать роли в PostgreSQL: изменение Secret не меняет пароль роли.
При выполненной диагностике Secret уже создан; повторно создавать его не нужно.

```bash
kubectl apply -f k8s/inventory-postgres.yaml
kubectl rollout status statefulset/inventory-postgres
# В отдельном терминале:
kubectl port-forward svc/inventory-postgres 15434:5432
# В корне проекта, с установленным goose:
python3 scripts/migrate-inventory-kind.py
kubectl apply -f k8s/inventory-deployment.yaml -f k8s/inventory-service.yaml
kubectl rollout status deployment/inventory
kubectl port-forward svc/inventory 15053:50053
```

В Postman создайте gRPC request к `localhost:15053`, отключите TLS,
импортируйте `proto/inventory.proto` (reflection в сервере не включён).
Выберите `inventory.InventoryService / GetInventoryByID` и отправьте:

```json
{"product_id":"diagnostic-nonexistent-20261101"}
```

Для отсутствующего товара ожидается `NOT_FOUND` / `inventory not found`:
это подтверждает работу gRPC и выполнение запроса к таблице PostgreSQL.
Для проверки записи вызовите `CreateInventory` с новым product_id:

```json
{"product_id":"postman-demo-1","product_name":"Demo","category_id":"demo","stock":"10","unit_price_minor":"100"}
```

Затем запросите этот product_id через `GetInventoryByID`; ожидается `OK`.
Повторное создание того же товара возвращает `ALREADY_EXISTS`.
Порт 50053 на хосте уже был занят, поэтому используем 15053.

## Диагностика 2026-11-01

PVC ждал provisioner; scheduler выбрал ноду, но получил
`PreBind VolumeBinding: binding volumes: context deadline exceeded`.
Provisioner завершился с timeout к `10.96.0.1:443`, затем containerd
возвращал `failed to reserve container name`. Нода Ready, без taints,
CPU requests 5%, память 290Mi из ~64Gi, свободно 156Gi диска.
StorageClass standard / rancher.io/local-path / WaitForFirstConsumer исправен.

Дополнительно CoreDNS и kube-proxy не могли читать API (`Unauthorized`),
API server сообщал `service account token is not valid yet`. Старые журналы
имеют время запуска позже времени завершения: часы переводились назад.
Текущие часы хоста синхронизированы NTP. Причина самого перевода не установлена.
Перезапуск provisioner создал PV и связал существующий PVC;
перезапуск CoreDNS и kube-proxy обновил токены и восстановил DNS/Services.
Inventory прежде использовал fallback 127.0.0.1:5434; теперь URL берётся из Secret.

Основные выполненные команды (чтение Secret выполнялось без вывода значений):

```bash
kubectl get nodes
kubectl get pods,pvc,svc -A -o wide
kubectl get events -A --sort-by=.metadata.creationTimestamp
kubectl describe node grpc-course-control-plane
kubectl describe pvc postgres-data-inventory-postgres-0
kubectl describe pod inventory-postgres-0
kubectl describe pod -n local-path-storage local-path-provisioner-75f7fc7dc5-847xw
kubectl logs -n kube-system kube-scheduler-grpc-course-control-plane
kubectl logs -n local-path-storage deployment/local-path-provisioner
kubectl logs -n kube-system deployment/coredns
kubectl logs -n kube-system daemonset/kube-proxy
kubectl logs -n kube-system kube-apiserver-grpc-course-control-plane
docker exec grpc-course-control-plane journalctl -u containerd --no-pager -n 60
docker exec grpc-course-control-plane df -h
timedatectl status
kubectl rollout restart deployment/local-path-provisioner -n local-path-storage
kubectl rollout restart deployment/coredns -n kube-system
kubectl rollout restart daemonset/kube-proxy -n kube-system
kubectl rollout restart deployment/inventory
```

Secret создан через временный JSON с правами 0600 вне репозитория;
файл затем удалён. Goose применил существующие миграции 001 и 002 до версии 2.
Кластер, PVC и данные не удалялись. Бизнес-логика Go не изменялась.

Итоговая проверка: PostgreSQL и Inventory `1/1 Running`, PVC `Bound` (1Gi).
Services: inventory-postgres `10.96.100.151:5432`, inventory `10.96.91.66:50053`.
Проверка реальным Go gRPC-клиентом через localhost:15053:
`GetInventoryByID` вернул ожидаемый `NotFound`; SQL подтвердил применённые
версии Goose 1 и 2. Проверка не добавляла тестовых данных.
