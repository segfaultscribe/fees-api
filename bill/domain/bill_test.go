package domain

import (
	"testing"
	"time"
)

func TestNewBill(t *testing.T) {
	createdAt := time.Now()
	bill := NewBill("bill-1", USD, createdAt)

	if bill.Status != BillOpen {
		t.Errorf("Status = %v, want %v", bill.Status, BillOpen)
	}
	if bill.LineItems == nil {
		t.Error("expected LineItems map to be initialised, got nil")
	}
	if bill.BillID != "bill-1" {
		t.Errorf("BillID = %q, want %q", bill.BillID, "bill-1")
	}
	if bill.Currency != USD {
		t.Errorf("Currency = %v, want %v", bill.Currency, USD)
	}
	if !bill.CreatedAt.Equal(createdAt) {
		t.Errorf("CreatedAt = %v, want %v", bill.CreatedAt, createdAt)
	}
	if !bill.ClosedAt.IsZero() {
		t.Errorf("ClosedAt = %v, want zero value", bill.ClosedAt)
	}
}
