# Локальные mTLS-сертификаты

[Локальный запуск](local-development.md) · [Архитектура](architecture.md)

Order читает `certs/ca.crt`, `certs/order-client.crt`, `certs/order-client.key`;
Payment — `certs/ca.crt`, `certs/payment-server.crt`, `certs/payment-server.key`.
Payment проверяет клиентский сертификат, Order проверяет серверный с
`ServerName=localhost`. Приватный ключ CA приложениям не нужен.

Для нового клона следующий Bash-блок создаёт собственный локальный CA и
сертификаты на 30 дней. Он остановится, если `certs/` уже существует, чтобы
сохранить существующие ключи.

```bash
(
  set -eu
  umask 077
  if [ -e certs ]; then
    echo 'certs/ already exists; certificate generation stopped' >&2
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
Docker build context. Сертификаты предназначены для локальной разработки.
