package agent

import (
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"sync"
	"time"

	"github.com/gamma-omg/trading-bot/internal/market"
	"github.com/shopspring/decimal"
)

type jsonReportBuilder struct {
	log    *slog.Logger
	report JsonReport
	spent  decimal.Decimal
	gained decimal.Decimal
	mu     sync.Mutex
}

type JsonReport struct {
	TotalGain    string                `json:"total_gain,omitempty"`
	TotalGainPct float64               `json:"total_gain_pct,omitempty"`
	Deals        map[string][]JsonDeal `json:"deals,omitempty"`
}

type JsonDeal struct {
	BuyTime  time.Time `json:"buy_time,omitzero,omitempty"`
	SellTime time.Time `json:"sell_time,omitzero,omitempty"`
	Spend    string    `json:"spend,omitempty"`
	Gain     string    `json:"gain,omitempty"`
	GainPct  float64   `json:"gain_pct,omitempty"`
}

func NewJsonReportBuilder(log *slog.Logger) *jsonReportBuilder {
	return &jsonReportBuilder{
		log: log,
		report: JsonReport{
			Deals: map[string][]JsonDeal{},
		},
	}
}

func (r *jsonReportBuilder) SubmitDeal(d market.Deal) {
	r.mu.Lock()
	defer r.mu.Unlock()

	pct := 0.0
	if !d.Spend.IsZero() {
		pct, _ = d.Gain.Div(d.Spend).Float64()
	}

	deals := r.report.Deals[d.Symbol]
	deals = append(deals, JsonDeal{
		BuyTime:  d.BuyTime,
		SellTime: d.SellTime,
		Spend:    d.Spend.String(),
		Gain:     d.Gain.String(),
		GainPct:  pct,
	})
	r.report.Deals[d.Symbol] = deals

	r.spent = r.spent.Add(d.Spend)
	r.gained = r.gained.Add(d.Gain)

	r.report.TotalGain = r.gained.String()

	if r.spent.IsZero() {
		return
	}

	pct, _ = r.gained.Div(r.spent).Float64()
	r.report.TotalGainPct = pct

	r.log.Info("deal closed", slog.String("symbol", d.Symbol), slog.Float64("gain_pct", pct), slog.Time("buy_time", d.BuyTime), slog.Time("sell_time", d.SellTime))
}

func (r *jsonReportBuilder) Write(w io.Writer) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	e := json.NewEncoder(w)
	if err := e.Encode(r.report); err != nil {
		return fmt.Errorf("failed to write trading report: %w", err)
	}

	return nil
}
