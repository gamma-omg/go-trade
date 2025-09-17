package indicator

import "fmt"

type tradingIndicator interface {
	GetSignal() (s Signal, err error)
}

type WeightedIndicator struct {
	Weight    float64
	Indicator tradingIndicator
}

type EnsembleIndicator struct {
	Children []WeightedIndicator
}

func (i *EnsembleIndicator) GetSignal() (s Signal, err error) {
	var act float64
	var totalWeight float64
	for _, c := range i.Children {
		var signal Signal
		signal, err = c.Indicator.GetSignal()
		if err != nil {
			err = fmt.Errorf("failed to get signal from one of the children: %w", err)
			return
		}

		act += float64(signal.Act) * signal.Confidence * c.Weight
		totalWeight += c.Weight
	}

	if act > 0 {
		return Signal{
			Act:        ACT_BUY,
			Confidence: act / totalWeight,
		}, nil
	}
	if act < 0 {
		return Signal{
			Act:        ACT_SELL,
			Confidence: -act / totalWeight,
		}, nil
	}

	return Signal{
		Act:        ACT_HOLD,
		Confidence: 1.0,
	}, nil
}
