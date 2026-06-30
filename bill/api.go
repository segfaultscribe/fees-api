package bill

import (
	"context"

	"encore.app/bill/domain"
	"encore.app/bill/workflow"
	"encore.dev/beta/errs"
	"encore.dev/rlog"
	"go.temporal.io/sdk/client"
)

// CreateBill starts a new billing period as a Temporal workflow.
// Calling this with an already-used BillID is idempotent and returns
// the existing bill's current state.
//
//encore:api public method=POST path=/bills
func (s *Service) CreateBill(ctx context.Context, req *CreateRequest) (*Response, error) {
	currency, err := domain.ParseCurrency(req.Currency)
	if err != nil {
		return nil, &errs.Error{
			Code:    errs.InvalidArgument,
			Message: err.Error(),
		}
	}

	if req.BillID == "" {
		return nil, &errs.Error{
			Code:    errs.InvalidArgument,
			Message: "bill_id required",
		}
	}

	run, err := s.temporalClient.ExecuteWorkflow(
		ctx,
		client.StartWorkflowOptions{
			ID:        req.BillID,
			TaskQueue: taskQueue,
		},
		workflow.BillWorkflow,
		&workflow.BillWorkflowParams{
			BillID:   req.BillID,
			Currency: currency,
		},
	)
	if err != nil {
		rlog.Error("workflow creation failed", "workflow id", req.BillID)
		return nil, &errs.Error{
			Code:    errs.Internal,
			Message: "faield to create bill",
		}
	}
	rlog.Info("workflow created", "workflow id", req.BillID)

	resp, err := s.temporalClient.QueryWorkflow(
		ctx,
		run.GetID(),
		run.GetRunID(),
		workflow.GetBillState,
	)
	if err != nil {
		rlog.Error("workflow query failed", "workflow id", req.BillID)
		return nil, &errs.Error{
			Code:    errs.Internal,
			Message: "failed to retrieve bill state",
		}
	}

	var invoice workflow.BillInvoice
	if err := resp.Get(&invoice); err != nil {
		return nil, &errs.Error{
			Code:    errs.Internal,
			Message: "failed to decode bill state",
		}
	}

	return toBillResponse(invoice), nil
}

// AddLineItem adds a line item to an open bill. Calling this with an
// already-used LineID is idempotent and returns the existing line item.
//
//encore:api public method=POST path=/bills/:billID/line-items
func (s *Service) AddLineItem(ctx context.Context, billID string, req *LineItemRequest) (*LineItemResponse, error) {
	currency, err := domain.ParseCurrency(req.Currency)
	if err != nil {
		return nil, &errs.Error{
			Code:    errs.InvalidArgument,
			Message: err.Error(),
		}
	}

	if req.LineID == "" {
		return nil, &errs.Error{
			Code:    errs.InvalidArgument,
			Message: "line_id required",
		}
	}

	amount, err := domain.ToMinorUnits(req.Amount)
	if err != nil {
		return nil, &errs.Error{
			Code:    errs.InvalidArgument,
			Message: err.Error(),
		}
	}

	li, err := domain.NewLineItem(
		req.LineID,
		req.Description,
		amount,
		currency,
	)
	if err != nil {
		return nil, &errs.Error{
			Code:    errs.InvalidArgument,
			Message: err.Error(),
		}
	}

	handle, err := s.temporalClient.UpdateWorkflow(
		ctx,
		client.UpdateWorkflowOptions{
			WorkflowID:   billID,
			UpdateName:   workflow.AddLineItem,
			Args:         []interface{}{*li},
			WaitForStage: client.WorkflowUpdateStageCompleted,
		},
	)
	if err != nil {
		return nil, mapWorkflowErr(err)
	}
	rlog.Info("new line item added", "line item id", req.LineID, "workflow id", billID)

	var result domain.LineItem
	if err := handle.Get(ctx, &result); err != nil {
		return nil, mapWorkflowErr(err)
	}

	return &LineItemResponse{
		LineID:      result.LineID,
		Description: result.Description,
		Amount:      domain.FromMinorUnits(result.Amount),
		Currency:    string(result.Currency),
	}, nil
}

// CloseBill closes an open bill, returning the final invoice and line
// item charges. Calling this on an already-closed bill is idempotent
// and returns the existing closed state.
//
//encore:api public method=POST path=/bills/:billID/close
func (s *Service) CloseBill(ctx context.Context, billID string) (*Response, error) {
	handle, err := s.temporalClient.UpdateWorkflow(
		ctx,
		client.UpdateWorkflowOptions{
			WorkflowID:   billID,
			UpdateName:   workflow.CloseBill,
			Args:         []interface{}{},
			WaitForStage: client.WorkflowUpdateStageCompleted,
		},
	)
	if err != nil {
		return nil, mapWorkflowErr(err)
	}
	rlog.Info("closed bill", "billID", billID)

	var invoice workflow.BillInvoice
	if err := handle.Get(ctx, &invoice); err != nil {
		return nil, mapWorkflowErr(err)
	}

	return toBillResponse(invoice), nil
}

// GetBillState returns the current state of a bill, whether open or closed.
//
//encore:api public method=GET path=/bills/:billID
func (s *Service) GetBillState(ctx context.Context, billID string) (*Response, error) {
	resp, err := s.temporalClient.QueryWorkflow(
		ctx,
		billID,
		"",
		workflow.GetBillState,
	)
	if err != nil {
		return nil, mapWorkflowErr(err)
	}

	var invoice workflow.BillInvoice
	if err := resp.Get(&invoice); err != nil {
		return nil, mapWorkflowErr(err)
	}

	return toBillResponse(invoice), nil
}
