// Package domain contains the core domain types and methods for the Fees API.
package domain

import "time"

// BillStatus indicates the Bill's current state.
// Currently supports open and closed.
type BillStatus string

// Supported Bill Statuses.
const (
	BillOpen   BillStatus = "OPEN"
	BillClosed BillStatus = "CLOSED"
)

// Bill represents the main workflow data.
type Bill struct {
	BillID    string
	Currency  Currency
	LineItems map[string]*LineItem
	Status    BillStatus
	CreatedAt time.Time
	ClosedAt  time.Time
}

// NewBill creates a new Bill instance.
// It accepts id and createdAt as parameters because Bill is constructed inside
// a Temporal workflow, where ID generation and time must come from the workflow's
// deterministic APIs rather than normal Go functions.
func NewBill(id string, currency Currency, createdAt time.Time) *Bill {
	return &Bill{
		BillID:    id,
		Currency:  currency,
		LineItems: make(map[string]*LineItem),
		Status:    BillOpen,
		CreatedAt: createdAt,
	}
}
