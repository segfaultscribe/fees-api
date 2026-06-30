package domain

import (
	"errors"
	"strings"
	"time"
)

// ErrDescriptionEmpty indicates the line item description is empty.
var (
	ErrDescriptionEmpty = errors.New("empty description")
	ErrLineIDEmpty      = errors.New("empty line_id")
)

// LineItem represents a single charge added to a Bill.
type LineItem struct {
	LineID      string
	Description string
	Amount      int64
	Currency    Currency
	CreatedAt   time.Time
}

// NewLineItem validates description and amount, then returns a new LineItem.
// A zero amount is supported (no-charge, free or complimentary items).
func NewLineItem(lineID, description string, amount int64, currency Currency) (*LineItem, error) {
	description = strings.TrimSpace(description)
	if lineID == "" {
		return nil, ErrLineIDEmpty
	}
	if description == "" {
		return nil, ErrDescriptionEmpty
	}
	if amount < 0 {
		return nil, ErrAmountNegative
	}

	return &LineItem{
		LineID:      lineID,
		Description: description,
		Amount:      amount,
		Currency:    currency,
		CreatedAt:   time.Now(),
	}, nil
}
