package indicator

import "fmt"

type Action int

const (
	ActBuy  Action = 1
	ActHold Action = 0
	ActSell Action = -1
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
