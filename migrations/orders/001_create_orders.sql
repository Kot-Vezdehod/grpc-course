-- +goose Up
CREATE TABLE orders (
    id TEXT PRIMARY KEY,
    currency TEXT NOT NULL
);

CREATE TABLE order_items (
    id BIGSERIAL PRIMARY KEY,
    order_id TEXT NOT NULL REFERENCES orders(id) ON DELETE CASCADE,
    product_id TEXT NOT NULL,
    quantity INTEGER NOT NULL CHECK (quantity > 0),
    unit_price_minor BIGINT NOT NULL CHECK (unit_price_minor >= 0)
);

CREATE INDEX order_items_order_id_idx ON order_items(order_id);

-- +goose Down
DROP TABLE order_items;
DROP TABLE orders;
