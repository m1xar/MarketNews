package db

import (
	"context"
	"database/sql"
)

func EnsureSchema(ctx context.Context, db *sql.DB) error {
	const ddl = `
CREATE EXTENSION IF NOT EXISTS pgcrypto;

CREATE TABLE IF NOT EXISTS ff_news (
	id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
	name TEXT NOT NULL,
	country TEXT NOT NULL,
	currency TEXT NOT NULL,
	news_key TEXT NULL,
	forecast_rate NUMERIC(5, 4) NULL,
	created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
	UNIQUE (name, country, currency)
);

ALTER TABLE ff_news
	ADD COLUMN IF NOT EXISTS news_key TEXT,
	ADD COLUMN IF NOT EXISTS forecast_rate NUMERIC(5, 4);

CREATE INDEX IF NOT EXISTS ff_news_key_idx ON ff_news (news_key);

CREATE TABLE IF NOT EXISTS ff_events (
	id BIGSERIAL PRIMARY KEY,
	news_id UUID NOT NULL REFERENCES ff_news(id) ON DELETE CASCADE,
	ff_id BIGINT NOT NULL,
	event_time TIMESTAMPTZ NULL,
	impact TEXT NOT NULL,
	actual_value DOUBLE PRECISION NULL,
	forecast_value DOUBLE PRECISION NULL,
	previous_value DOUBLE PRECISION NULL,
	surprise DOUBLE PRECISION NULL,
	z_score DOUBLE PRECISION NULL,
	metadata JSONB NULL,
	created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
	UNIQUE (news_id, ff_id)
);

ALTER TABLE ff_events
	ADD COLUMN IF NOT EXISTS surprise DOUBLE PRECISION,
	ADD COLUMN IF NOT EXISTS z_score DOUBLE PRECISION;

CREATE TABLE IF NOT EXISTS ff_event_type_stats (
	news_id UUID PRIMARY KEY REFERENCES ff_news(id) ON DELETE CASCADE,
	sigma_surprise DOUBLE PRECISION NULL,
	mean_surprise DOUBLE PRECISION NULL,
	n_samples INTEGER NOT NULL DEFAULT 0,
	updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS ff_event_asset_stats (
	news_id UUID NOT NULL REFERENCES ff_news(id) ON DELETE CASCADE,
	asset_symbol TEXT NOT NULL,
	delta_minutes INTEGER NOT NULL,
	beta DOUBLE PRECISION NULL,
	alpha DOUBLE PRECISION NULL,
	r2 DOUBLE PRECISION NULL,
	n_samples INTEGER NOT NULL DEFAULT 0,
	p_pos_given_zpos DOUBLE PRECISION NULL,
	p_neg_given_zneg DOUBLE PRECISION NULL,
	p_dir DOUBLE PRECISION NULL,
	mean_abs_return DOUBLE PRECISION NULL,
	updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
	PRIMARY KEY (news_id, asset_symbol, delta_minutes)
);

ALTER TABLE ff_event_asset_stats
	DROP COLUMN IF EXISTS impact_score;

CREATE INDEX IF NOT EXISTS ff_event_asset_stats_asset_delta_idx ON ff_event_asset_stats (asset_symbol, delta_minutes);

CREATE TABLE IF NOT EXISTS ff_event_asset_returns (
	event_id BIGINT NOT NULL REFERENCES ff_events(id) ON DELETE CASCADE,
	asset_symbol TEXT NOT NULL,
	delta_minutes INTEGER NOT NULL,
	price_0 DOUBLE PRECISION NULL,
	price_delta DOUBLE PRECISION NULL,
	return_ln DOUBLE PRECISION NULL,
	created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
	PRIMARY KEY (event_id, asset_symbol, delta_minutes)
);

CREATE INDEX IF NOT EXISTS ff_event_asset_returns_asset_delta_idx ON ff_event_asset_returns (asset_symbol, delta_minutes);

CREATE TABLE IF NOT EXISTS ff_upcoming_analysis (
	event_id BIGINT PRIMARY KEY,
	news_id UUID NOT NULL REFERENCES ff_news(id) ON DELETE CASCADE,
	ff_id BIGINT NOT NULL,
	event_time TIMESTAMPTZ NULL,
	country TEXT NOT NULL,
	currency TEXT NOT NULL,
	symbol TEXT NOT NULL,
	importance TEXT NOT NULL,
	forecast_value DOUBLE PRECISION NULL,
	previous_value DOUBLE PRECISION NULL,
	metadata JSONB NULL,
	analysis_text TEXT NOT NULL,
	created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
	updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

ALTER TABLE ff_upcoming_analysis
	DROP CONSTRAINT IF EXISTS ff_upcoming_analysis_event_id_fkey;

CREATE INDEX IF NOT EXISTS ff_upcoming_analysis_event_time_idx ON ff_upcoming_analysis (event_time);
CREATE INDEX IF NOT EXISTS ff_upcoming_analysis_news_id_idx ON ff_upcoming_analysis (news_id);

CREATE TABLE IF NOT EXISTS trade_analysis (
	trade_id BIGINT PRIMARY KEY,
	pair_name TEXT NOT NULL,
	entry_price DOUBLE PRECISION NOT NULL,
	amount DOUBLE PRECISION NOT NULL,
	asset TEXT NOT NULL,
	direction TEXT NOT NULL,
	stop_loss DOUBLE PRECISION NULL,
	take_profit DOUBLE PRECISION NULL,
	open_date TIMESTAMPTZ NOT NULL,
	current_ts TIMESTAMPTZ NOT NULL,
	current_price DOUBLE PRECISION NOT NULL,
	events_date DATE NOT NULL,
	analysis_text TEXT NOT NULL,
	created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
	updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
`
	_, err := db.ExecContext(ctx, ddl)
	return err
}
