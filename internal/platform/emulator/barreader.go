package emulator

import (
	"bufio"
	"encoding/csv"
	"fmt"
	"io"
	"os"
	"strconv"
	"time"

	"github.com/gamma-omg/trading-bot/internal/market"
	"github.com/shopspring/decimal"
)

type barFilter func(b market.Bar) bool

type barReader struct {
	rdr    *csv.Reader
	filter barFilter
	bars   chan market.Bar
}

func newBarReader(dataPath string) (*barReader, error) {
	return newBarReaderWithFilter(dataPath, func(b market.Bar) bool { return true })
}

func newBarReaderWithFilter(dataPath string, filter barFilter) (*barReader, error) {
	f, err := os.Open(dataPath)
	if err != nil {
		return nil, fmt.Errorf("unable to create bar streamer: %w", err)
	}

	streamer := &barReader{
		rdr:    csv.NewReader(bufio.NewReader(f)),
		filter: filter,
		bars:   make(chan market.Bar, 64),
	}
	return streamer, nil
}

func (b *barReader) Read() error {
	if _, err := b.rdr.Read(); err != nil {
		return fmt.Errorf("failed to read csv header: %w", err)
	}

	for {
		data, err := b.rdr.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("failed to read bar data: %w", err)
		}

		timestamp, err := strconv.ParseFloat(data[0], 64)
		if err != nil {
			return fmt.Errorf("failed to parse bar time: %w", err)
		}

		open, err := decimal.NewFromString(data[1])
		if err != nil {
			return fmt.Errorf("failed to read oepn price: %w", err)
		}

		high, err := decimal.NewFromString(data[2])
		if err != nil {
			return fmt.Errorf("failed to read high price: %w", err)
		}

		low, err := decimal.NewFromString(data[3])
		if err != nil {
			return fmt.Errorf("failed to read low price: %w", err)
		}

		close, err := decimal.NewFromString(data[4])
		if err != nil {
			return fmt.Errorf("failed to read close price: %w", err)
		}

		volume, err := decimal.NewFromString(data[5])
		if err != nil {
			return fmt.Errorf("failed to read volume price: %w", err)
		}

		bar := market.Bar{
			Time:   time.Unix(int64(timestamp), 0),
			Open:   open,
			Close:  close,
			High:   high,
			Low:    low,
			Volume: volume,
		}
		if b.filter(bar) {
			b.bars <- bar
		}
	}

	close(b.bars)
	return nil
}
