package queries

const (
	ListEventsForSurpriseBase = `
SELECT e.id, e.news_id, e.actual_value, e.forecast_value
FROM ff_events e
JOIN ff_news n ON n.id = e.news_id
WHERE actual_value IS NOT NULL AND forecast_value IS NOT NULL
`
	ListEventsForSurpriseByNewsID = `
SELECT e.id, e.news_id, e.actual_value, e.forecast_value
FROM ff_events e
WHERE e.news_id = $1 AND e.actual_value IS NOT NULL AND e.forecast_value IS NOT NULL
ORDER BY e.id ASC;
`
	UpdateEventSurprise = `
UPDATE ff_events
SET surprise = $2,
	z_score = $3
WHERE id = $1;
`
	UpsertEventTypeStats = `
INSERT INTO ff_event_type_stats (
	news_id, sigma_surprise, mean_surprise, n_samples, updated_at
) VALUES (
	$1, $2, $3, $4, now()
)
ON CONFLICT (news_id) DO UPDATE SET
	sigma_surprise = EXCLUDED.sigma_surprise,
	mean_surprise = EXCLUDED.mean_surprise,
	n_samples = EXCLUDED.n_samples,
	updated_at = EXCLUDED.updated_at;
`
	ListEventsForAssetStatsBase = `
SELECT e.id, e.news_id, e.event_time, e.z_score, n.currency
FROM ff_events e
JOIN ff_news n ON n.id = e.news_id
WHERE e.z_score IS NOT NULL AND e.event_time IS NOT NULL
`
	UpsertEventAssetStats = `
INSERT INTO ff_event_asset_stats (
	news_id, asset_symbol, delta_minutes, beta, alpha, r2, n_samples,
	p_pos_given_zpos, p_neg_given_zneg, p_dir, mean_abs_return, updated_at
) VALUES (
	$1, $2, $3, $4, $5, $6, $7,
	$8, $9, $10, $11, now()
)
ON CONFLICT (news_id, asset_symbol, delta_minutes) DO UPDATE SET
	beta = EXCLUDED.beta,
	alpha = EXCLUDED.alpha,
	r2 = EXCLUDED.r2,
	n_samples = EXCLUDED.n_samples,
	p_pos_given_zpos = EXCLUDED.p_pos_given_zpos,
	p_neg_given_zneg = EXCLUDED.p_neg_given_zneg,
	p_dir = EXCLUDED.p_dir,
	mean_abs_return = EXCLUDED.mean_abs_return,
	updated_at = EXCLUDED.updated_at;
`
	UpsertEventAssetReturn = `
INSERT INTO ff_event_asset_returns (
	event_id, asset_symbol, delta_minutes, price_0, price_delta, return_ln, created_at
) VALUES (
	$1, $2, $3, $4, $5, $6, now()
)
ON CONFLICT (event_id, asset_symbol, delta_minutes) DO UPDATE SET
	price_0 = EXCLUDED.price_0,
	price_delta = EXCLUDED.price_delta,
	return_ln = EXCLUDED.return_ln;
`
	UpdateNewsForecastRateBase = `
UPDATE ff_news n
SET forecast_rate = sub.rate
FROM (
	SELECT
		e.news_id,
		CASE
			WHEN COUNT(*) FILTER (WHERE e.actual_value IS NOT NULL AND e.forecast_value IS NOT NULL) = 0
				THEN NULL
			ELSE
				ROUND(
					COUNT(*) FILTER (WHERE e.actual_value IS NOT NULL AND e.forecast_value IS NOT NULL AND e.actual_value = e.forecast_value)::NUMERIC
					/
					COUNT(*) FILTER (WHERE e.actual_value IS NOT NULL AND e.forecast_value IS NOT NULL),
					4
				)
		END AS rate
	FROM ff_events e
	JOIN ff_news n ON n.id = e.news_id
	WHERE 1 = 1
`
	UpdateNewsForecastRateTail = `
	GROUP BY e.news_id
) AS sub
WHERE n.id = sub.news_id;
`
	UpdateNewsForecastRateByIDs = `
UPDATE ff_news n
SET forecast_rate = sub.rate
FROM (
	SELECT
		e.news_id,
		CASE
			WHEN COUNT(*) FILTER (WHERE e.actual_value IS NOT NULL AND e.forecast_value IS NOT NULL) = 0
				THEN NULL
			ELSE
				ROUND(
					COUNT(*) FILTER (WHERE e.actual_value IS NOT NULL AND e.forecast_value IS NOT NULL AND e.actual_value = e.forecast_value)::NUMERIC
					/
					COUNT(*) FILTER (WHERE e.actual_value IS NOT NULL AND e.forecast_value IS NOT NULL),
					4
				)
		END AS rate
	FROM ff_events e
	WHERE e.news_id = ANY($1::uuid[])
	GROUP BY e.news_id
) AS sub
WHERE n.id = sub.news_id;
`
)
