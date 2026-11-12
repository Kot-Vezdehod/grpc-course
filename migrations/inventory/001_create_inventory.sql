-- +goose Up

CREATE TABLE inventory
(
    product_id       TEXT PRIMARY KEY,
    product_name     TEXT   NOT NULL,
    category_id      TEXT   NOT NULL,
    stock            BIGINT NOT NULL CHECK (stock >= 0),
    unit_price_minor BIGINT NOT NULL CHECK (unit_price_minor > 0)
);


-- +goose Down
DROP TABLE inventory;