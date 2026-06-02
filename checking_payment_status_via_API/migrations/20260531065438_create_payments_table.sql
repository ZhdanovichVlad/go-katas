-- +goose Up
-- +goose StatementBegin
CREATE TABLE payments (
    id            BIGSERIAL PRIMARY KEY,
    order_id      TEXT        NOT NULL,
    vendor_tx_id  TEXT        NOT NULL,
    amount_cents  BIGINT      NOT NULL,
    status        TEXT        NOT NULL,
    paid_at       TIMESTAMPTZ NULL,
    is_locked     BOOLEAN     NOT NULL DEFAULT FALSE,
    time_locked   TIMESTAMPTZ NULL,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
);
CREATE UNIQUE INDEX idx_payments_vendor_tx_id ON payments (vendor_tx_id);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS idx_payments_status_updated_at;
DROP TABLE IF EXISTS payments;
-- +goose StatementEnd
