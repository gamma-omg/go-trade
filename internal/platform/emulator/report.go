package emulator

import (
	"encoding/json"
	"fmt"
	"io"
	"time"

	"github.com/shopspring/decimal"
)

type jsonReportBuilder struct {
	report JsonReport
	spent  decimal.Decimal
	gained decimal.Decimal
}

type JsonReport struct {
	TotalSpend   string                `json:"total_spend,omitempty"`
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

func newJsonReportBuilder() *jsonReportBuilder {
	return &jsonReportBuilder{
		report: JsonReport{
			Deals: map[string][]JsonDeal{},
		},
	}
}

func (r *jsonReportBuilder) SubmitDeal(d Deal) {
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

	r.report.TotalSpend = r.spent.String()
	r.report.TotalGain = r.gained.String()

	if r.spent.IsZero() {
		return
	}

	pct, _ = r.gained.Div(r.spent).Float64()
	r.report.TotalGainPct = pct
}

func (r *jsonReportBuilder) Write(w io.Writer) error {
	e := json.NewEncoder(w)
	if err := e.Encode(r.report); err != nil {
		return fmt.Errorf("failed to write trading report: %w", err)
	}

	return nil
}
