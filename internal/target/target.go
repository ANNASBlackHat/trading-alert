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

type TargetType string

const (
	TargetTypePrice    TargetType = "price"
	TargetTypeTrailing TargetType = "trailing"
)

type Target struct {
	ID          string    `json:"id"`
	BotName     string    `json:"bot_name,omitempty"`
	Symbol      string    `json:"symbol"`
	TargetPrice float64   `json:"target_price,omitempty"` // For standard price alerts
	Direction   Direction `json:"direction"`              // "up" (trailing rise) or "down" (trailing drop)
	CreatedAt   time.Time `json:"created_at"`
	LastState   State     `json:"last_state"`

	// Trailing Stop fields
	Type            TargetType `json:"type"`                       // "price" or "trailing"
	TrailingPercent float64    `json:"trailing_percent,omitempty"` // e.g. 15.0 for 15%
	TrailingValue   float64    `json:"trailing_value,omitempty"`   // e.g. 50.0 for $50
	ActivationPrice float64    `json:"activation_price,omitempty"` // optional threshold before tracking
	IsActive        bool       `json:"is_active"`                  // tracks if the trailing stop is currently active
	ExtremePrice    float64    `json:"extreme_price,omitempty"`    // peak (for down) or trough (for up)

	// Additional metadata
	Note string `json:"note,omitempty"` // optional message/label to include in alerts
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
