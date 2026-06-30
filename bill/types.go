package bill

import "time"

// CreateRequest is the request payload for creating a new bill.
type CreateRequest struct {
	BillID   string `json:"bill_id"`
	Currency string `json:"currency"`
}

// Response is the response payload representing a bill's current state.
type Response struct {
	BillID       string             `json:"bill_id"`
	Currency     string             `json:"currency"`
	Status       string             `json:"status"`
	CreatedAt    time.Time          `json:"created_at"`
	ClosedAt     *time.Time         `json:"closed_at,omitempty"`
	LineItems    []LineItemResponse `json:"line_items"`
	TotalInvoice string             `json:"total_invoice"`
}

// LineItemRequest is the request payload for adding a line item to a bill.
type LineItemRequest struct {
	LineID      string `json:"line_id"`
	Description string `json:"description"`
	Amount      string `json:"amount"`
	Currency    string `json:"currency"`
}

// LineItemResponse is the response shape for a single line item.
type LineItemResponse struct {
	LineID      string `json:"line_id"`
	Description string `json:"description"`
	Amount      string `json:"amount"`
	Currency    string `json:"currency"`
}
