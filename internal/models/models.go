package models

import "time"

type Failure struct {
	Date       string
	StatusCode int
	Err        string
}

type EventRow struct {
	ID        int64
	DateTime  *string
	Country   string
	Currency  string
	Impact    string
	Name      string
	Actual    string
	Forecast  string
	Previous  string
	Metadata  *string
	NewsKey   *string
	ActualF   *float64
	ForecastF *float64
	PreviousF *float64
}

type NewsRecord struct {
	ID           string
	Name         string
	Country      string
	Currency     string
	NewsKey      *string
	ForecastRate *float64
	CreatedAt    time.Time
}

type EventRecord struct {
	ID          int64
	NewsID      string
	FFID        int64
	EventTime   *time.Time
	Impact      string
	ActualValue *float64
	ForecastVal *float64
	PreviousVal *float64
	Surprise    *float64
	ZScore      *float64
	Metadata    *string
	CreatedAt   time.Time
}

type EventWithNews struct {
	Event EventRecord
	News  NewsRecord
}

type EventTypeStats struct {
	NewsID        string
	SigmaSurprise *float64
	MeanSurprise  *float64
	NSamples      int
	UpdatedAt     time.Time
}

type EventAssetStats struct {
	NewsID        string
	AssetSymbol   string
	DeltaMinutes  int
	Beta          *float64
	Alpha         *float64
	R2            *float64
	NSamples      int
	PPosGivenZPos *float64
	PNegGivenZNeg *float64
	PDir          *float64
	MeanAbsReturn *float64
	UpdatedAt     time.Time
}

type SurpriseEvent struct {
	ID       int64
	NewsID   string
	Actual   float64
	Forecast float64
}

type AssetEvent struct {
	ID       int64
	NewsID   string
	EventAt  time.Time
	ZScore   float64
	Currency string
}

type UpcomingEvent struct {
	EventID       int64
	FFID          int64
	EventTime     *time.Time
	Impact        string
	ForecastValue *float64
	PreviousValue *float64
	Metadata      *string
	News          NewsRecord
}

type EventAssetReturn struct {
	EventID      int64
	AssetSymbol  string
	DeltaMinutes int
	Price0       *float64
	PriceDelta   *float64
	ReturnLn     *float64
}

type EventHistoryItem struct {
	EventID       int64
	EventTime     time.Time
	ActualValue   *float64
	ForecastValue *float64
	PreviousValue *float64
	Surprise      *float64
	ZScore        *float64
	Returns       []EventAssetReturn
}

type TradeAnalysisRecord struct {
	TradeID      int64
	PairName     string
	EntryPrice   float64
	Amount       float64
	Asset        string
	Direction    string
	StopLoss     *float64
	TakeProfit   *float64
	OpenDate     time.Time
	CurrentDate  time.Time
	CurrentPrice float64
	EventsDate   time.Time
	AnalysisText string
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

type UpcomingAnalysisRecord struct {
	EventID       int64
	NewsID        string
	FFID          int64
	EventTime     *time.Time
	Country       string
	Currency      string
	Symbol        string
	Importance    string
	ForecastValue *float64
	PreviousValue *float64
	Metadata      *string
	AnalysisText  string
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

type UpcomingAnalysisDetail struct {
	UpcomingAnalysisRecord
	NewsName     string
	NewsKey      *string
	ForecastRate *float64
}

type NewsQuery struct {
	IDs         []string
	Names       []string
	Countries   []string
	Currencies  []string
	NewsKeys    []string
	CreatedFrom *time.Time
	CreatedTo   *time.Time
	Limit       int
	Offset      int
}

type EventQuery struct {
	IDs              []int64
	NewsIDs          []string
	FFIDs            []int64
	EventFrom        *time.Time
	EventTo          *time.Time
	Impacts          []string
	ActualMin        *float64
	ActualMax        *float64
	ForecastMin      *float64
	ForecastMax      *float64
	PreviousMin      *float64
	PreviousMax      *float64
	CreatedFrom      *time.Time
	CreatedTo        *time.Time
	MetadataEquals   *string
	MetadataContains *string

	NewsNames      []string
	NewsCountries  []string
	NewsCurrencies []string
	NewsKeys       []string

	Limit  int
	Offset int
}
