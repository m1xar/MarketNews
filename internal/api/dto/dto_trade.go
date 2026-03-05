package dto

type TradeAnalysisRequest struct {
	TradeID      int64    `json:"trade_id"`
	PairName     string   `json:"pair"`
	EntryPrice   float64  `json:"entry_price"`
	Amount       float64  `json:"amount"`
	Asset        string   `json:"asset"`
	Direction    string   `json:"direction"`
	StopLoss     *float64 `json:"stop_loss"`
	TakeProfit   *float64 `json:"take_profit"`
	OpenDate     string   `json:"open_date"`
	CurrentDate  string   `json:"current_date"`
	CurrentPrice float64  `json:"current_price"`
	EventsDate   string   `json:"events_date"`
}
