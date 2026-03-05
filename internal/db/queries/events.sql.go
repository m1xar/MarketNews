package queries

const (
	SelectEventByID = `
SELECT id, news_id, ff_id, event_time, impact, actual_value, forecast_value, previous_value, surprise, z_score, metadata, created_at
FROM ff_events
WHERE id = $1;
`
	InsertEvent = `
INSERT INTO ff_events (
	news_id, ff_id, event_time, impact, actual_value, forecast_value, previous_value, surprise, z_score, metadata
) VALUES (
	$1, $2, $3, $4, $5, $6, $7, $8, $9, $10::jsonb
)
RETURNING id, news_id, ff_id, event_time, impact, actual_value, forecast_value, previous_value, surprise, z_score, metadata, created_at;
`
	UpdateEvent = `
UPDATE ff_events
SET news_id = $2,
	ff_id = $3,
	event_time = $4,
	impact = $5,
	actual_value = $6,
	forecast_value = $7,
	previous_value = $8,
	surprise = $9,
	z_score = $10,
	metadata = $11::jsonb
WHERE id = $1
RETURNING id, news_id, ff_id, event_time, impact, actual_value, forecast_value, previous_value, surprise, z_score, metadata, created_at;
`
	DeleteEvent = `
DELETE FROM ff_events
WHERE id = $1;
`
	UpsertEvent = `
INSERT INTO ff_events (
	news_id, ff_id, event_time, impact, actual_value, forecast_value, previous_value, surprise, z_score, metadata
) VALUES (
	$1, $2, $3, $4, $5, $6, $7, $8, $9, $10::jsonb
)
ON CONFLICT (news_id, ff_id) DO UPDATE SET
	event_time = EXCLUDED.event_time,
	impact = EXCLUDED.impact,
	actual_value = EXCLUDED.actual_value,
	forecast_value = EXCLUDED.forecast_value,
	previous_value = EXCLUDED.previous_value,
	surprise = EXCLUDED.surprise,
	z_score = EXCLUDED.z_score,
	metadata = EXCLUDED.metadata;
`
	ListEventsBase = `
SELECT
	e.id, e.news_id, e.ff_id, e.event_time, e.impact, e.actual_value, e.forecast_value, e.previous_value, e.surprise, e.z_score, e.metadata, e.created_at,
	n.id, n.name, n.country, n.currency, n.news_key, n.forecast_rate, n.created_at
FROM ff_events e
JOIN ff_news n ON n.id = e.news_id
`
)
