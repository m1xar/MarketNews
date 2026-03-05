package queries

const (
	TruncateAll = `
TRUNCATE ff_event_asset_returns, ff_event_asset_stats, ff_event_type_stats, ff_events, ff_news RESTART IDENTITY CASCADE;
`
)
