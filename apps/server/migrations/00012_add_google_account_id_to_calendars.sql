-- +goose Up
ALTER TABLE calendars
    ADD COLUMN google_account_id TEXT;

CREATE INDEX idx_calendars_google_account_id ON calendars (google_account_id);
CREATE INDEX idx_calendars_user_account ON calendars (user_id, google_account_id);

UPDATE calendars c
SET google_account_id = uga.id::text
FROM user_google_accounts uga
WHERE c.google_account_id IS NULL
  AND c.user_id = uga.user_id::text
  AND uga.is_primary = TRUE;

-- +goose Down
DROP INDEX IF EXISTS idx_calendars_user_account;
DROP INDEX IF EXISTS idx_calendars_google_account_id;

ALTER TABLE calendars
    DROP COLUMN IF EXISTS google_account_id;
