package money

import (
	"errors"
	"fmt"

	"github.com/shopspring/decimal"
)

// 2 decimal places precision
const scale = 2

type Money struct {
	d decimal.Decimal
}

var Zero = Money{}

func Parse(s string) (Money, error) {
	d, err := decimal.NewFromString(s)
	if err != nil {
		return Money{}, fmt.Errorf("invalid amount %q", s)
	}
	return fromDecimal(d)
}

func fromDecimal(d decimal.Decimal) (Money, error) {
	if !d.Equal(d.Truncate(scale)) {
		return Money{}, fmt.Errorf("amount %s has more than %d decimal places", d, scale)
	}
	return Money{d}, nil
}

func (m *Money) UnmarshalJSON(data []byte) error {
	s := string(data)
	switch {
	case s == "null":
		return errors.New("amount is required")
	case len(s) > 0 && s[0] == '"':
		return errors.New("amount must be a JSON number, not a string")
	}
	d, err := decimal.NewFromString(s)
	if err != nil {
		return fmt.Errorf("invalid amount %s", s)
	}
	parsed, err := fromDecimal(d)
	if err != nil {
		return err
	}
	*m = parsed
	return nil
}

func (m Money) MarshalJSON() ([]byte, error) {
	return []byte(m.d.StringFixed(scale)), nil
}

func (m Money) IsPositive() bool {
	return m.d.IsPositive()
}

func (m Money) Equal(other Money) bool {
	return m.d.Equal(other.d)
}

func (m Money) Add(other Money) Money {
	return Money{m.d.Add(other.d)}
}

func (m Money) DivInt(n int) Money {
	if n == 0 {
		return Zero
	}
	return Money{m.d.DivRound(decimal.NewFromInt(int64(n)), scale)}
}

func (m Money) String() string {
	return m.d.String()
}