-- 008_outbox.sql
-- Modul 11: Transactional Outbox + Notifications

-- ============================================================
-- Outbox: antrian event yang ditulis DALAM transaksi bisnis.
-- published_at NULL = belum direlay ke Redis/Asynq.
-- ============================================================
CREATE TABLE IF NOT EXISTS outbox (
    id            BIGSERIAL PRIMARY KEY,
    event_type    VARCHAR(50)  NOT NULL,
    payload       JSONB        NOT NULL,
    created_at    TIMESTAMP    NOT NULL DEFAULT NOW(),
    published_at  TIMESTAMP
);

-- Partial index: hanya baris yang belum dipublish (relay membaca ini).
CREATE INDEX IF NOT EXISTS idx_outbox_unpublished
    ON outbox (id)
    WHERE published_at IS NULL;

-- ============================================================
-- Notifications: catatan notifikasi yang sudah dikirim.
-- outbox_id UNIQUE = idempotency key (satu event → satu notifikasi).
-- ============================================================
CREATE TABLE IF NOT EXISTS notifications (
    id         BIGSERIAL PRIMARY KEY,
    outbox_id  BIGINT      NOT NULL UNIQUE REFERENCES outbox(id) ON DELETE CASCADE,
    user_id    INT         NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    channel    VARCHAR(20) NOT NULL DEFAULT 'email',
    subject    VARCHAR(200) NOT NULL,
    body       TEXT        NOT NULL,
    sent_at    TIMESTAMP   NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_notifications_user_id ON notifications (user_id);

-- ============================================================
-- Permission untuk memantau antrian (hanya admin).
-- ============================================================
INSERT INTO permissions (name, description) VALUES
  ('queue:monitor', 'Memantau antrian background job & outbox')
ON CONFLICT (name) DO NOTHING;

INSERT INTO role_permissions (role_name, permission_name) VALUES
  ('admin', 'queue:monitor')
ON CONFLICT DO NOTHING;
