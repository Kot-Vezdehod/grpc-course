-- +goose Up

CREATE TABLE inventory_reservations
(
    order_id   TEXT PRIMARY KEY,
    product_id TEXT NOT NULL REFERENCES inventory(product_id),
    quantity   BIGINT NOT NULL CHECK (quantity > 0)
);


-- +goose Down
DROP TABLE inventory_reservations;