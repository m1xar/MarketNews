package app

import (
	"context"
	"database/sql"
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"

	"MarketNews/internal/marketdata"
	"MarketNews/internal/models"
)

type StatsConfig struct {
	EventFrom     *time.Time
	EventTo       *time.Time
	Names         []string
	Countries     []string
	Currencies    []string
	DeltasMinutes []int
}

type StatsService struct {
	txBeginner TxBeginner
	repo       StatsRepository
	market     MarketDataClient
}

func NewStatsService(txBeginner TxBeginner, repo StatsRepository, market MarketDataClient) *StatsService {
	return &StatsService{
		txBeginner: txBeginner,
		repo:       repo,
		market:     market,
	}
}

func (s *StatsService) ComputeStats(ctx context.Context, cfg StatsConfig) error {
	if len(cfg.DeltasMinutes) == 0 {
		return fmt.Errorf("deltas_minutes is required")
	}

	if err := s.computeSurpriseStats(ctx, cfg); err != nil {
		return err
	}
	if err := s.computeForecastRate(ctx, cfg); err != nil {
		return err
	}
	if err := s.computeAssetStats(ctx, cfg); err != nil {
		return err
	}
	return nil
}

func (s *StatsService) ComputeStatsForNewsIDs(ctx context.Context, newsIDs []string, cfg StatsConfig) error {
	if len(newsIDs) == 0 {
		return nil
	}
	if len(cfg.DeltasMinutes) == 0 {
		return fmt.Errorf("deltas_minutes is required")
	}

	for _, newsID := range newsIDs {
		if strings.TrimSpace(newsID) == "" {
			continue
		}
		if err := s.computeSurpriseStatsForNewsID(ctx, newsID); err != nil {
			return err
		}
	}

	_, err := s.repo.UpdateNewsForecastRateByNewsIDs(ctx, newsIDs)
	if err != nil {
		return fmt.Errorf("update forecast rate by news ids: %w", err)
	}
	for _, newsID := range newsIDs {
		if strings.TrimSpace(newsID) == "" {
			continue
		}
		if err := s.ComputeAssetStatsForNewsID(ctx, newsID, cfg); err != nil {
			return err
		}
	}
	return nil
}

func (s *StatsService) ComputeAssetStatsForNewsID(ctx context.Context, newsID string, cfg StatsConfig) error {
	if strings.TrimSpace(newsID) == "" {
		return fmt.Errorf("news_id is required")
	}
	if len(cfg.DeltasMinutes) == 0 {
		return fmt.Errorf("deltas_minutes is required")
	}

	events, err := s.repo.ListAssetEventsByNewsID(ctx, newsID)
	if err != nil {
		return fmt.Errorf("list events for asset stats by news_id: %w", err)
	}
	if len(events) == 0 {
		return nil
	}

	maxDelta, err := maxDeltaMinutes(cfg.DeltasMinutes)
	if err != nil {
		return err
	}

	if err := s.processAssetNews(ctx, cfg, maxDelta, newsID, events); err != nil {
		return err
	}
	return nil
}

func (s *StatsService) computeSurpriseStats(ctx context.Context, cfg StatsConfig) error {
	events, err := s.repo.ListEventsForSurprise(ctx, cfg.EventFrom, cfg.EventTo, cfg.Names, cfg.Countries, cfg.Currencies)
	if err != nil {
		return fmt.Errorf("list events for surprise: %w", err)
	}
	byNews := map[string][]models.SurpriseEvent{}
	for _, ev := range events {
		byNews[ev.NewsID] = append(byNews[ev.NewsID], ev)
	}

	for newsID, list := range byNews {
		if err := s.computeSurpriseForGroup(ctx, newsID, list); err != nil {
			return err
		}
	}
	return nil
}

func (s *StatsService) computeSurpriseStatsForNewsID(ctx context.Context, newsID string) error {
	events, err := s.repo.ListEventsForSurpriseByNewsID(ctx, newsID)
	if err != nil {
		return fmt.Errorf("list events for surprise by news_id: %w", err)
	}
	if len(events) == 0 {
		return nil
	}
	return s.computeSurpriseForGroup(ctx, newsID, events)
}

func (s *StatsService) computeSurpriseForGroup(ctx context.Context, newsID string, events []models.SurpriseEvent) error {
	n := len(events)
	if n == 0 {
		return nil
	}
	surprises := make([]float64, 0, n)
	sum := 0.0
	for _, ev := range events {
		sVal := ev.Actual - ev.Forecast
		surprises = append(surprises, sVal)
		sum += sVal
	}
	mean := sum / float64(n)
	var sigma *float64
	if n >= 2 {
		varSum := 0.0
		for _, s := range surprises {
			diff := s - mean
			varSum += diff * diff
		}
		if varSum > 0 {
			v := math.Sqrt(varSum / float64(n-1))
			if v > 0 {
				sigma = &v
			}
		}
	}
	meanPtr := &mean

	tx, err := s.txBeginner.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}

	for i, ev := range events {
		sVal := surprises[i]
		var z *float64
		if sigma != nil && *sigma > 0 {
			zVal := sVal / *sigma
			z = &zVal
		}
		if err := s.repo.UpdateEventSurprise(ctx, tx, ev.ID, &sVal, z); err != nil {
			_ = tx.Rollback()
			return fmt.Errorf("update surprise: %w", err)
		}
	}
	if err := s.repo.UpsertEventTypeStats(ctx, tx, newsID, sigma, meanPtr, n); err != nil {
		_ = tx.Rollback()
		return fmt.Errorf("upsert event type stats: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit surprise stats: %w", err)
	}
	return nil
}

func (s *StatsService) computeForecastRate(ctx context.Context, cfg StatsConfig) error {
	_, err := s.repo.UpdateNewsForecastRate(ctx, cfg.EventFrom, cfg.EventTo, cfg.Names, cfg.Countries, cfg.Currencies)
	if err != nil {
		return fmt.Errorf("update forecast rate: %w", err)
	}
	return nil
}

func (s *StatsService) computeAssetStats(ctx context.Context, cfg StatsConfig) error {
	events, err := s.repo.ListEventsForAssetStats(ctx, cfg.EventFrom, cfg.EventTo, cfg.Names, cfg.Countries, cfg.Currencies)
	if err != nil {
		return fmt.Errorf("list events for asset stats: %w", err)
	}
	if len(events) == 0 {
		return nil
	}

	maxDelta, err := maxDeltaMinutes(cfg.DeltasMinutes)
	if err != nil {
		return err
	}

	var currentNewsID string
	var buffer []models.AssetEvent
	for _, ev := range events {
		if currentNewsID == "" {
			currentNewsID = ev.NewsID
		}
		if ev.NewsID != currentNewsID {
			if err := s.processAssetNews(ctx, cfg, maxDelta, currentNewsID, buffer); err != nil {
				return err
			}
			buffer = buffer[:0]
			currentNewsID = ev.NewsID
		}
		buffer = append(buffer, ev)
	}
	if len(buffer) > 0 {
		if err := s.processAssetNews(ctx, cfg, maxDelta, currentNewsID, buffer); err != nil {
			return err
		}
	}
	return nil
}

type assetGroupKey struct {
	Asset string
	Delta int
}

type assetGroupData struct {
	Z           []float64
	R           []float64
	SumAbsR     float64
	CountZPos   int
	CountZNeg   int
	CountZPosRp int
	CountZNegRn int
}

type assetEventReturn struct {
	EventID int64
	Asset   string
	Delta   int
	P0      float64
	P1      float64
	R       float64
}

type assetResult struct {
	key        assetGroupKey
	beta       *float64
	alpha      *float64
	r2         *float64
	nSamples   int
	pPosGivenZ *float64
	pNegGivenZ *float64
	pDir       *float64
	meanAbs    *float64
}

func (s *StatsService) processAssetNews(
	ctx context.Context,
	cfg StatsConfig,
	maxDelta int,
	newsID string,
	newsEvents []models.AssetEvent,
) error {
	if len(newsEvents) == 0 {
		return nil
	}
	groups := map[assetGroupKey]*assetGroupData{}
	returns := make([]assetEventReturn, 0, len(newsEvents)*len(cfg.DeltasMinutes))

	for _, ev := range newsEvents {
		asset := assetSymbolForCurrency(ev.Currency)
		t0 := ev.EventAt.UTC().Truncate(time.Minute)
		start := t0.Add(-time.Minute)
		end := t0.Add(time.Duration(maxDelta-1) * time.Minute)
		if end.Before(start) {
			end = start
		}

		var series marketdata.CandleSeries
		var dxyPairs []dxyPairSeries
		var err error
		if asset == "DXY" {
			dxyPairs, err = s.fetchDXYSeries(ctx, start, end)
			if err != nil {
				continue
			}
		} else {
			series, err = s.market.GetSeriesRange(ctx, asset, start, end)
			if err != nil {
				continue
			}
		}

		p0, ok0 := priceAt(asset, series, dxyPairs, t0)
		if !ok0 || p0 <= 0 {
			continue
		}

		for _, delta := range cfg.DeltasMinutes {
			if delta <= 0 {
				continue
			}
			t1 := t0.Add(time.Duration(delta) * time.Minute)
			p1, ok1 := priceAt(asset, series, dxyPairs, t1)
			if !ok1 || p1 <= 0 {
				continue
			}
			r := math.Log(p1 / p0)

			returns = append(returns, assetEventReturn{
				EventID: ev.ID,
				Asset:   asset,
				Delta:   delta,
				P0:      p0,
				P1:      p1,
				R:       r,
			})

			key := assetGroupKey{Asset: asset, Delta: delta}
			g, ok := groups[key]
			if !ok {
				g = &assetGroupData{}
				groups[key] = g
			}
			updateAssetGroup(g, ev.ZScore, r)
		}
	}

	results := computeAssetResults(groups)

	tx, err := s.txBeginner.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	if err := s.upsertAssetReturns(ctx, tx, returns); err != nil {
		_ = tx.Rollback()
		return err
	}
	if err := s.upsertAssetStats(ctx, tx, newsID, results); err != nil {
		_ = tx.Rollback()
		return err
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit asset stats: %w", err)
	}
	return nil
}

func priceAt(asset string, series marketdata.CandleSeries, dxyPairs []dxyPairSeries, t time.Time) (float64, bool) {
	if asset == "DXY" {
		return dxyAtTime(dxyPairs, t)
	}
	return series.PriceAt(t)
}

func updateAssetGroup(g *assetGroupData, z, r float64) {
	g.Z = append(g.Z, z)
	g.R = append(g.R, r)
	g.SumAbsR += math.Abs(r)
	if z > 0 {
		g.CountZPos++
		if r > 0 {
			g.CountZPosRp++
		}
	} else if z < 0 {
		g.CountZNeg++
		if r < 0 {
			g.CountZNegRn++
		}
	}
}

func computeAssetResults(groups map[assetGroupKey]*assetGroupData) []assetResult {
	results := make([]assetResult, 0, len(groups))
	for key, g := range groups {
		n := len(g.Z)
		if n == 0 {
			continue
		}
		sumZ, sumR, sumZ2, sumZR := 0.0, 0.0, 0.0, 0.0
		for i := 0; i < n; i++ {
			z := g.Z[i]
			r := g.R[i]
			sumZ += z
			sumR += r
			sumZ2 += z * z
			sumZR += z * r
		}
		zBar := sumZ / float64(n)
		rBar := sumR / float64(n)
		varZ := sumZ2 - float64(n)*zBar*zBar

		var beta *float64
		var alpha *float64
		if varZ > 0 {
			b := (sumZR - float64(n)*zBar*rBar) / varZ
			a := rBar - b*zBar
			beta = &b
			alpha = &a
		}

		var r2 *float64
		if beta != nil {
			ssRes := 0.0
			ssTot := 0.0
			for i := 0; i < n; i++ {
				z := g.Z[i]
				r := g.R[i]
				rHat := *alpha + *beta*z
				diff := r - rHat
				ssRes += diff * diff
				dev := r - rBar
				ssTot += dev * dev
			}
			if ssTot > 0 {
				val := 1 - (ssRes / ssTot)
				r2 = &val
			}
		}

		var pPos *float64
		if g.CountZPos > 0 {
			v := float64(g.CountZPosRp) / float64(g.CountZPos)
			pPos = &v
		}
		var pNeg *float64
		if g.CountZNeg > 0 {
			v := float64(g.CountZNegRn) / float64(g.CountZNeg)
			pNeg = &v
		}

		var pDir *float64
		if beta != nil {
			countDir := 0
			for i := 0; i < n; i++ {
				if sign(g.R[i]) == sign(*beta*g.Z[i]) {
					countDir++
				}
			}
			v := float64(countDir) / float64(n)
			pDir = &v
		}

		meanAbs := g.SumAbsR / float64(n)
		meanAbsPtr := &meanAbs

		results = append(results, assetResult{
			key:        key,
			beta:       beta,
			alpha:      alpha,
			r2:         r2,
			nSamples:   n,
			pPosGivenZ: pPos,
			pNegGivenZ: pNeg,
			pDir:       pDir,
			meanAbs:    meanAbsPtr,
		})
	}
	return results
}

func (s *StatsService) upsertAssetReturns(ctx context.Context, tx *sql.Tx, returns []assetEventReturn) error {
	for _, ret := range returns {
		p0 := ret.P0
		p1 := ret.P1
		r := ret.R
		if err := s.repo.UpsertEventAssetReturn(
			ctx,
			tx,
			ret.EventID,
			ret.Asset,
			ret.Delta,
			&p0,
			&p1,
			&r,
		); err != nil {
			return fmt.Errorf("upsert event asset return: %w", err)
		}
	}
	return nil
}

func (s *StatsService) upsertAssetStats(ctx context.Context, tx *sql.Tx, newsID string, results []assetResult) error {
	for _, res := range results {
		if err := s.repo.UpsertEventAssetStats(
			ctx,
			tx,
			newsID,
			res.key.Asset,
			res.key.Delta,
			res.beta,
			res.alpha,
			res.r2,
			res.nSamples,
			res.pPosGivenZ,
			res.pNegGivenZ,
			res.pDir,
			res.meanAbs,
		); err != nil {
			return fmt.Errorf("upsert event asset stats: %w", err)
		}
	}
	return nil
}

func assetSymbolForCurrency(currency string) string {
	c := strings.ToUpper(strings.TrimSpace(currency))
	if c == "USD" {
		return "DXY"
	}
	return c + "/USD"
}

type dxyPairSeries struct {
	Symbol string
	Power  float64
	Series marketdata.CandleSeries
}

func (s *StatsService) fetchDXYSeries(ctx context.Context, start, end time.Time) ([]dxyPairSeries, error) {
	pairs := []struct {
		Symbol string
		Power  float64
	}{
		{Symbol: "EUR/USD", Power: -0.576},
		{Symbol: "USD/JPY", Power: 0.136},
		{Symbol: "GBP/USD", Power: -0.119},
		{Symbol: "USD/CAD", Power: 0.091},
		{Symbol: "USD/SEK", Power: 0.042},
		{Symbol: "USD/CHF", Power: 0.036},
	}

	out := make([]dxyPairSeries, 0, len(pairs))
	for _, p := range pairs {
		series, err := s.market.GetSeriesRange(ctx, p.Symbol, start, end)
		if err != nil {
			return nil, err
		}
		out = append(out, dxyPairSeries{
			Symbol: p.Symbol,
			Power:  p.Power,
			Series: series,
		})
	}
	return out, nil
}

func dxyAtTime(pairs []dxyPairSeries, t time.Time) (float64, bool) {
	dxy := 50.14348112
	for _, p := range pairs {
		price, ok := p.Series.PriceAt(t)
		if !ok || price <= 0 {
			return 0, false
		}
		dxy *= math.Pow(price, p.Power)
	}
	return dxy, true
}

func maxDeltaMinutes(deltas []int) (int, error) {
	if len(deltas) == 0 {
		return 0, fmt.Errorf("deltas_minutes is required")
	}
	max := deltas[0]
	for _, v := range deltas {
		if v > max {
			max = v
		}
	}
	if max <= 0 {
		return 0, fmt.Errorf("deltas_minutes must include a positive value")
	}
	return max, nil
}

func sign(v float64) int {
	if v > 0 {
		return 1
	}
	if v < 0 {
		return -1
	}
	return 0
}

func fmtTime(t *time.Time) string {
	if t == nil {
		return "all"
	}
	return t.UTC().Format(time.RFC3339)
}

func fmtFloat(v float64) string {
	return strconv.FormatFloat(v, 'f', 8, 64)
}

func fmtFloatPtr(v *float64) string {
	if v == nil {
		return "null"
	}
	return strconv.FormatFloat(*v, 'f', 8, 64)
}
