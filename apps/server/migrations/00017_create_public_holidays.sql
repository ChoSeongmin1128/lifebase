-- +goose Up
CREATE TABLE public_holidays_kr (
    locdate DATE NOT NULL,
    name TEXT NOT NULL,
    year INT NOT NULL,
    month INT NOT NULL,
    date_kind TEXT,
    is_holiday BOOLEAN NOT NULL DEFAULT TRUE,
    fetched_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (locdate, name)
);

CREATE INDEX idx_public_holidays_kr_year_month ON public_holidays_kr (year, month);
CREATE INDEX idx_public_holidays_kr_locdate ON public_holidays_kr (locdate);

CREATE TABLE public_holiday_sync_state (
    year INT NOT NULL,
    month INT NOT NULL,
    last_synced_at TIMESTAMPTZ NOT NULL,
    last_result_code TEXT,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (year, month)
);

-- +goose Down
DROP TABLE IF EXISTS public_holiday_sync_state;
DROP INDEX IF EXISTS idx_public_holidays_kr_locdate;
DROP INDEX IF EXISTS idx_public_holidays_kr_year_month;
DROP TABLE IF EXISTS public_holidays_kr;
