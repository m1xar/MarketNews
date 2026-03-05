package queries

const (
	ListUpcomingEvents = `
SELECT
	e.id, e.ff_id, e.event_time, e.impact, e.forecast_value, e.previous_value, e.metadata,
	n.id, n.name, n.country, n.currency, n.news_key, n.forecast_rate, n.created_at
FROM ff_events e
JOIN ff_news n ON n.id = e.news_id
WHERE e.event_time IS NOT NULL
	AND e.event_time >= $1
	AND e.event_time <= $2
ORDER BY e.event_time ASC, e.id ASC;
`
	SelectEventTypeStatsByNewsID = `
SELECT news_id, sigma_surprise, mean_surprise, n_samples, updated_at
FROM ff_event_type_stats
WHERE news_id = $1;
`
	ListAssetStatsByNewsID = `
SELECT news_id, asset_symbol, delta_minutes, beta, alpha, r2, n_samples,
	p_pos_given_zpos, p_neg_given_zneg, p_dir, mean_abs_return, updated_at
FROM ff_event_asset_stats
WHERE news_id = $1
ORDER BY asset_symbol ASC, delta_minutes ASC;
`
	ListEventTypeStatsByNewsIDs = `
SELECT news_id, sigma_surprise, mean_surprise, n_samples, updated_at
FROM ff_event_type_stats
WHERE news_id = ANY($1);
`
	ListAssetStatsByNewsIDs = `
SELECT news_id, asset_symbol, delta_minutes, beta, alpha, r2, n_samples,
	p_pos_given_zpos, p_neg_given_zneg, p_dir, mean_abs_return, updated_at
FROM ff_event_asset_stats
WHERE news_id = ANY($1)
ORDER BY news_id ASC, asset_symbol ASC, delta_minutes ASC;
`
	ListRecentEventsByNewsID = `
SELECT id, event_time, actual_value, forecast_value, previous_value, surprise, z_score
FROM ff_events
WHERE news_id = $1
	AND event_time IS NOT NULL
	AND actual_value IS NOT NULL
ORDER BY event_time DESC, id DESC
LIMIT $2;
`
	ListEventsByNewsID = `
SELECT id, event_time, actual_value, forecast_value, previous_value, surprise, z_score
FROM ff_events
WHERE news_id = $1
	AND event_time IS NOT NULL
	AND actual_value IS NOT NULL
ORDER BY event_time DESC, id DESC;
`
	ListReturnsByEventIDs = `
SELECT event_id, asset_symbol, delta_minutes, price_0, price_delta, return_ln
FROM ff_event_asset_returns
WHERE event_id = ANY($1)
ORDER BY event_id ASC, asset_symbol ASC, delta_minutes ASC;
`
	UpsertUpcomingAnalysis = `
INSERT INTO ff_upcoming_analysis (
	event_id, news_id, ff_id, event_time,
	country, currency, symbol, importance,
	forecast_value, previous_value, metadata, analysis_text,
	created_at, updated_at
) VALUES (
	$1, $2, $3, $4,
	$5, $6, $7, $8,
	$9, $10, $11::jsonb, $12,
	now(), now()
)
ON CONFLICT (event_id) DO UPDATE SET
	news_id = EXCLUDED.news_id,
	ff_id = EXCLUDED.ff_id,
	event_time = EXCLUDED.event_time,
	country = EXCLUDED.country,
	currency = EXCLUDED.currency,
	symbol = EXCLUDED.symbol,
	importance = EXCLUDED.importance,
	forecast_value = EXCLUDED.forecast_value,
	previous_value = EXCLUDED.previous_value,
	metadata = EXCLUDED.metadata,
	analysis_text = EXCLUDED.analysis_text,
	updated_at = now();
`
	SelectUpcomingAnalysisByEventID = `
SELECT
	a.event_id, a.news_id, a.ff_id, a.event_time,
	a.country, a.currency, a.symbol, a.importance,
	a.forecast_value, a.previous_value, a.metadata, a.analysis_text,
	a.created_at, a.updated_at,
	n.name, n.news_key, n.forecast_rate
FROM ff_upcoming_analysis a
JOIN ff_news n ON n.id = a.news_id
WHERE a.event_id = $1;
`
	ListUpcomingAnalysisByRange = `
SELECT
	a.event_id, a.news_id, a.ff_id, a.event_time,
	a.country, a.currency, a.symbol, a.importance,
	a.forecast_value, a.previous_value, a.metadata, a.analysis_text,
	a.created_at, a.updated_at,
	n.name, n.news_key, n.forecast_rate
FROM ff_upcoming_analysis a
JOIN ff_news n ON n.id = a.news_id
WHERE a.event_time IS NOT NULL
	AND a.event_time >= $1
	AND a.event_time <= $2
ORDER BY a.event_time ASC, a.event_id ASC;
`
	TruncateUpcomingAnalysis = `
TRUNCATE TABLE ff_upcoming_analysis;
`
)
