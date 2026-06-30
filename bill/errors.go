package bill

import (
	"errors"

	"encore.app/bill/workflow"
	"encore.dev/beta/errs"
	"go.temporal.io/api/serviceerror"
)

// bill/errors.go

// mapWorkflowErr is a helper that translates errors returned from Temporal workflow
// interactions into appropriate Encore API errors.
func mapWorkflowErr(err error) error {
	switch {
	case errors.Is(err, workflow.ErrBillClosed):
		return &errs.Error{Code: errs.FailedPrecondition, Message: "bill is closed"}
	case errors.Is(err, workflow.ErrCurrencyMismatch):
		return &errs.Error{Code: errs.FailedPrecondition, Message: "currency mismatch"}
	case errors.Is(err, workflow.ErrLineIDEmpty):
		return &errs.Error{Code: errs.InvalidArgument, Message: "line_id is required"}
	case errors.Is(err, workflow.ErrBillIDEmpty):
		return &errs.Error{Code: errs.InvalidArgument, Message: "bill_id is required"}
	default:
		var notFoundErr *serviceerror.NotFound
		if errors.As(err, &notFoundErr) {
			return &errs.Error{Code: errs.NotFound, Message: "bill not found"}
		}
		return &errs.Error{Code: errs.Internal, Message: "workflow operation failed"}
	}
}
