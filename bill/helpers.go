package bill

import (
	"time"

	"encore.app/bill/domain"
	"encore.app/bill/workflow"
)

func toBillResponse(inv workflow.BillInvoice) *Response {
	items := make([]LineItemResponse, 0, len(inv.LineItems))
	for _, li := range inv.LineItems {
		items = append(items, LineItemResponse{
			LineID:      li.LineID,
			Description: li.Description,
			Amount:      domain.FromMinorUnits(li.Amount),
			Currency:    string(li.Currency),
		})
	}

	var closedAt *time.Time
	if !inv.ClosedAt.IsZero() {
		closedAt = &inv.ClosedAt
	}

	return &Response{
		BillID:       inv.BillID,
		Currency:     string(inv.Currency),
		Status:       string(inv.Status),
		CreatedAt:    inv.CreatedAt,
		ClosedAt:     closedAt,
		LineItems:    items,
		TotalInvoice: domain.FromMinorUnits(inv.TotalInvoice),
	}
}
