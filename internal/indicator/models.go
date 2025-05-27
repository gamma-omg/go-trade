package indicator

type Action int

const (
	ACT_BUY  Action = 1
	ACT_HOLD Action = 0
	ACT_SELL Action = -1
)

type Signal struct {
	Act        Action
	Confidence float32
}
