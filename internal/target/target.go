package target

import "time"

type Direction string

type State string

const (
	DirectionUp   Direction = "up"
	DirectionDown Direction = "down"

	StateUnknown State = "unknown"
	StateAbove   State = "above"
	StateBelow   State = "below"
)

type Target struct {
	ID          string    `json:"id"`
	BotName     string    `json:"bot_name,omitempty"`
	Symbol      string    `json:"symbol"`
	TargetPrice float64   `json:"target_price"`
	Direction   Direction `json:"direction"`
	CreatedAt   time.Time `json:"created_at"`
	LastState   State     `json:"last_state"`
}

type Store interface {
	Save(target Target) (Target, error)
	Get(id string) (Target, error)
	ListAll() ([]Target, error)
	ListBySymbol(symbol string) ([]Target, error)
	ListByBotName(botName string) ([]Target, error)
	Update(target Target) error
	Delete(id string) error
}

func NormalizeState(price, targetPrice float64) State {
	if price >= targetPrice {
		return StateAbove
	}
	return StateBelow
}

func IsValidDirection(direction string) bool {
	return direction == string(DirectionUp) || direction == string(DirectionDown)
}
