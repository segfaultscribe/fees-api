// Package workflow contains the durable execution logic using temporal
package workflow

import (
	"errors"

	"encore.app/bill/domain"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

// TaskQueue is the Temporal task queue used by the bill worker.
const TaskQueue = "fees-api-bill-queue"

// BillWorkflowParams are the parameters passed to BillWorkflow.
type BillWorkflowParams struct {
	Currency domain.Currency `json:"currency"`
	BillID   string          `json:"bill_id"`
}

// BillInvoice represents the queryable summary of a Bill, including
// the computed total invoice amount.
type BillInvoice struct {
	domain.Bill
	TotalInvoice int64
}

// Update and Query handler names registered on the BillWorkflow.
const (
	CloseBill    = "close_bill"
	AddLineItem  = "add_line_item"
	GetBillState = "get_bill_state"
)

var (
	// ErrCurrencyMismatch indicates a line item's currency does not match the bill's currency.
	ErrCurrencyMismatch = errors.New("currency mismatch")

	// ErrBillClosed indicates the bill is closed when performing an operation that requires bill to be open.
	ErrBillClosed = errors.New("bill closed")

	// ErrLineIDEmpty indicates attempt to add line item using empty LineID.
	ErrLineIDEmpty = errors.New("line id cannot be empty")

	// ErrBillIDEmpty indicates attempt to start workflow using an empty BillID.
	ErrBillIDEmpty = errors.New("bill id cannot be empty")
)

// BillWorkflow is the core long running durable execution process.
// It opens a Bill and keeps it open unless closed via a closing update.
func BillWorkflow(ctx workflow.Context, billParams *BillWorkflowParams) error {
	logger := workflow.GetLogger(ctx)

	if billParams.BillID == "" {
		logger.Error("bill workflow started with empty BillID")
		return temporal.NewApplicationError(ErrBillIDEmpty.Error(), "BillIDEmpty")
	}

	bill := domain.NewBill(
		billParams.BillID,
		billParams.Currency,
		workflow.Now(ctx),
	)

	logger.Info("bill created", "billID", billParams.BillID)

	if err := workflow.SetUpdateHandlerWithOptions(
		ctx,
		AddLineItem,
		func(ctx workflow.Context, li domain.LineItem) (domain.LineItem, error) {
			if bill.Status == domain.BillClosed {
				return domain.LineItem{}, temporal.NewApplicationError(ErrBillClosed.Error(), "BillClosed")
			}
			if existing, ok := bill.LineItems[li.LineID]; ok {
				return existing, nil
			}
			bill.LineItems[li.LineID] = li
			logger.Info("line item added", "lineID", li.LineID, "billID", bill.BillID)
			return li, nil
		},
		workflow.UpdateHandlerOptions{
			Validator: func(ctx workflow.Context, li domain.LineItem) error {
				if bill.Status == domain.BillClosed {
					return temporal.NewApplicationError(ErrBillClosed.Error(), "BillClosed")
				}
				if li.LineID == "" {
					return temporal.NewApplicationError(ErrLineIDEmpty.Error(), "LineIDEmpty")
				}
				if bill.Currency != li.Currency {
					return temporal.NewApplicationError(ErrCurrencyMismatch.Error(), "CurrencyMismatch")
				}
				return nil
			},
		},
	); err != nil {
		logger.Error("register add line item update handler failed", "error", err)
		return err
	}

	if err := workflow.SetUpdateHandlerWithOptions(
		ctx,
		CloseBill,
		func(ctx workflow.Context) (BillInvoice, error) {
			if bill.Status != domain.BillClosed {
				bill.Status = domain.BillClosed
				bill.ClosedAt = workflow.Now(ctx)
				logger.Info("bill closed", "billID", bill.BillID)
			}
			return BillInvoice{
				Bill:         *bill,
				TotalInvoice: calculateTotal(bill.LineItems),
			}, nil
		},
		workflow.UpdateHandlerOptions{},
	); err != nil {
		logger.Error("register close bill update handler failed", "error", err)
		return err
	}

	if err := workflow.SetQueryHandlerWithOptions(
		ctx,
		GetBillState,
		func() (BillInvoice, error) {
			total := calculateTotal(bill.LineItems)
			return BillInvoice{
				Bill:         *bill,
				TotalInvoice: total,
			}, nil
		},
		workflow.QueryHandlerOptions{},
	); err != nil {
		logger.Error("register get bill state query handler failed", "error", err)
		return err
	}

	if err := workflow.Await(ctx, func() bool {
		return bill.Status == domain.BillClosed
	}); err != nil {
		logger.Error("workflow runner await failed", "error", err)
		return err
	}

	if err := workflow.Await(ctx, func() bool {
		return workflow.AllHandlersFinished(ctx)
	}); err != nil {
		logger.Error("workflow finish handler await failed", "error", err)
		return err
	}
	logger.Info("workflow exited")
	return nil
}

// calculateTotal sums the amounts of all line items.
func calculateTotal(items map[string]domain.LineItem) int64 {
	var total int64
	for _, item := range items {
		total += item.Amount
	}
	return total
}
