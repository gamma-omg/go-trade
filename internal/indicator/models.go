package indicator

import "fmt"

type Action int

const (
	ACT_BUY  Action = 1
	ACT_HOLD Action = 0
	ACT_SELL Action = -1
)

type Signal struct {
	Act        Action
	Confidence float64
}

func (a Action) String() string {
	switch a {
	case 1:
		return "ACT_BUY"
	case 0:
		return "ACT_HOLD"
	case -1:
		return "ACT_SELL"
	default:
		return fmt.Sprintf("ACT_%d", a)
	}
}
