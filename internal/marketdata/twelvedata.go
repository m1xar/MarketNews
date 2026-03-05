package marketdata

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"
)

type Candle struct {
	Time  time.Time
	Close float64
}

type CandleSeries struct {
	Times  []time.Time
	Close  []float64
	ByTime map[int64]float64
}

type TwelveDataClient struct {
	APIKey  string
	Client  *http.Client
	BaseURL string
}

func NewTwelveDataClient(apiKey string, client *http.Client) *TwelveDataClient {
	if client == nil {
		client = &http.Client{Timeout: 20 * time.Second}
	}
	return &TwelveDataClient{
		APIKey:  apiKey,
		Client:  client,
		BaseURL: "https://api.twelvedata.com/time_series",
	}
}

func (c *TwelveDataClient) GetSeriesRange(ctx context.Context, symbol string, start, end time.Time) (CandleSeries, error) {
	symbol = strings.TrimSpace(symbol)
	if symbol == "" {
		return CandleSeries{}, fmt.Errorf("empty symbol")
	}
	return c.fetchSeries(ctx, symbol, start, end)
}

func (c *TwelveDataClient) fetchSeries(ctx context.Context, symbol string, start, end time.Time) (CandleSeries, error) {
	q := url.Values{}
	q.Set("symbol", symbol)
	const tdInterval = time.Minute
	q.Set("interval", "1min")
	q.Set("start_date", start.UTC().Format("2006-01-02 15:04:05"))
	q.Set("end_date", end.UTC().Format("2006-01-02 15:04:05"))
	q.Set("timezone", "UTC")
	q.Set("apikey", c.APIKey)
	reqURL := c.BaseURL + "?" + q.Encode()

	doReq := func() (*http.Response, error) {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
		if err != nil {
			return nil, err
		}
		return c.Client.Do(req)
	}

	resp, err := doReq()
	if err != nil {
		if isTimeout(err) {
			time.Sleep(1 * time.Minute)
			resp, err = doReq()
		}
	}
	if err != nil {
		return CandleSeries{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusTooManyRequests {
		time.Sleep(1 * time.Minute)
		resp.Body.Close()
		resp, err = doReq()
		if err != nil {
			return CandleSeries{}, err
		}
		defer resp.Body.Close()
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return CandleSeries{}, fmt.Errorf("twelvedata status %d", resp.StatusCode)
	}

	var payload tdSeriesResponse
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return CandleSeries{}, err
	}
	if payload.Status == "error" {
		if payload.Message == "" {
			payload.Message = "twelvedata error"
		}
		return CandleSeries{}, errors.New(payload.Message)
	}

	if len(payload.Values) == 0 {
		return CandleSeries{ByTime: map[int64]float64{}}, nil
	}

	candles := make([]Candle, 0, len(payload.Values))
	for _, v := range payload.Values {
		t, err := parseTDTime(v.DateTime)
		if err != nil {
			continue
		}
		closeVal, err := strconv.ParseFloat(strings.TrimSpace(v.Close), 64)
		if err != nil {
			continue
		}
		// TwelveData timestamps are candle open time; shift to close time.
		candles = append(candles, Candle{Time: t.UTC().Add(tdInterval), Close: closeVal})
	}
	sort.Slice(candles, func(i, j int) bool { return candles[i].Time.Before(candles[j].Time) })

	series := CandleSeries{
		Times:  make([]time.Time, 0, len(candles)),
		Close:  make([]float64, 0, len(candles)),
		ByTime: map[int64]float64{},
	}
	for _, c := range candles {
		series.Times = append(series.Times, c.Time)
		series.Close = append(series.Close, c.Close)
		series.ByTime[c.Time.Unix()] = c.Close
	}
	return series, nil
}

func (c CandleSeries) PriceAt(t time.Time) (float64, bool) {
	if len(c.ByTime) == 0 {
		return 0, false
	}
	v, ok := c.ByTime[t.UTC().Unix()]
	return v, ok
}

func isTimeout(err error) bool {
	var netErr net.Error
	if errors.As(err, &netErr) && netErr.Timeout() {
		return true
	}
	return false
}

type tdSeriesResponse struct {
	Status  string           `json:"status"`
	Message string           `json:"message"`
	Values  []tdSeriesCandle `json:"values"`
}

type tdSeriesCandle struct {
	DateTime string `json:"datetime"`
	Close    string `json:"close"`
}

func parseTDTime(s string) (time.Time, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return time.Time{}, fmt.Errorf("empty time")
	}
	if t, err := time.Parse("2006-01-02 15:04:05", s); err == nil {
		return t.UTC(), nil
	}
	if t, err := time.Parse("2006-01-02 15:04", s); err == nil {
		return t.UTC(), nil
	}
	return time.Parse(time.RFC3339, s)
}
