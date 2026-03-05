package queries

const (
	SelectNewsByKey = `
SELECT id, name, country, currency, news_key, forecast_rate, created_at
FROM ff_news
WHERE news_key = $1;
`
	SelectNewsByID = `
SELECT id, name, country, currency, news_key, forecast_rate, created_at
FROM ff_news
WHERE id = $1;
`
	SelectNewsByNatural = `
SELECT id, name, country, currency, news_key, forecast_rate, created_at
FROM ff_news
WHERE name = $1 AND country = $2 AND currency = $3;
`
	InsertNews = `
INSERT INTO ff_news (name, country, currency, news_key)
VALUES ($1, $2, $3, $4)
RETURNING id, name, country, currency, news_key, forecast_rate, created_at;
`
	UpdateNews = `
UPDATE ff_news
SET name = $2, country = $3, currency = $4, news_key = $5
WHERE id = $1
RETURNING id, name, country, currency, news_key, forecast_rate, created_at;
`
	DeleteNews = `
DELETE FROM ff_news
WHERE id = $1;
`
	UpsertNewsByNatural = `
INSERT INTO ff_news (name, country, currency, news_key)
VALUES ($1, $2, $3, $4)
ON CONFLICT (name, country, currency) DO UPDATE SET
	news_key = COALESCE(ff_news.news_key, EXCLUDED.news_key)
RETURNING id;
`
	ListNewsBase = `
SELECT id, name, country, currency, news_key, forecast_rate, created_at
FROM ff_news
`
)
