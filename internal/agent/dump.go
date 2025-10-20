package agent

import (
	"encoding/csv"
	"fmt"
	"io"
	"strconv"

	"github.com/gamma-omg/trading-bot/internal/market"
)

type csvBarsDump struct {
	w           *csv.Writer
	writeHeader bool
}

func newCsvBarsDump(w io.Writer) *csvBarsDump {
	return &csvBarsDump{csv.NewWriter(w), true}
}

func (d *csvBarsDump) Dump(bar market.Bar) error {
	if d.writeHeader {
		if err := d.w.Write([]string{"timestamp", "open", "high", "low", "close", "volume"}); err != nil {
			return fmt.Errorf("failed to write bars dump csv header: %w", err)
		}
		d.writeHeader = false
	}

	err := d.w.Write([]string{
		strconv.FormatInt(bar.Time.Unix(), 10),
		bar.Open.String(),
		bar.High.String(),
		bar.Low.String(),
		bar.Close.String(),
		bar.Volume.String()})

	if err != nil {
		return fmt.Errorf("failed to dump bar: %w", err)
	}

	d.w.Flush()
	return d.w.Error()
}
