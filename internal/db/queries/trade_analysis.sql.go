package queries

const (
	UpsertTradeAnalysis = `
INSERT INTO trade_analysis (
	trade_id,
	pair_name,
	entry_price,
	amount,
	asset,
	direction,
	stop_loss,
	take_profit,
	open_date,
	current_ts,
	current_price,
	events_date,
	analysis_text,
	created_at,
	updated_at
) VALUES (
	$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, now(), now()
)
ON CONFLICT (trade_id) DO UPDATE SET
	pair_name = EXCLUDED.pair_name,
	entry_price = EXCLUDED.entry_price,
	amount = EXCLUDED.amount,
	asset = EXCLUDED.asset,
	direction = EXCLUDED.direction,
	stop_loss = EXCLUDED.stop_loss,
	take_profit = EXCLUDED.take_profit,
	open_date = EXCLUDED.open_date,
	current_ts = EXCLUDED.current_ts,
	current_price = EXCLUDED.current_price,
	events_date = EXCLUDED.events_date,
	analysis_text = EXCLUDED.analysis_text,
	updated_at = now();
`
	SelectTradeAnalysisByID = `
SELECT
	trade_id,
	pair_name,
	entry_price,
	amount,
	asset,
	direction,
	stop_loss,
	take_profit,
	open_date,
	current_ts,
	current_price,
	events_date,
	analysis_text,
	created_at,
	updated_at
FROM trade_analysis
WHERE trade_id = $1;
`
)
