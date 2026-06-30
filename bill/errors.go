package bill

import (
	"errors"

	"encore.dev/beta/errs"
	"go.temporal.io/api/serviceerror"
	"go.temporal.io/sdk/temporal"
)

// mapWorkflowErr is a helper that translates errors returned from Temporal workflow
// interactions into appropriate Encore API errors.
func mapWorkflowErr(err error) error {
	var appErr *temporal.ApplicationError
	if errors.As(err, &appErr) {
		switch appErr.Type() {
		case "BillClosed":
			return &errs.Error{Code: errs.FailedPrecondition, Message: "bill is closed"}
		case "CurrencyMismatch":
			return &errs.Error{Code: errs.FailedPrecondition, Message: "currency mismatch"}
		case "LineIDEmpty":
			return &errs.Error{Code: errs.InvalidArgument, Message: "line_id is required"}
		case "BillIDEmpty":
			return &errs.Error{Code: errs.InvalidArgument, Message: "bill_id is required"}
		}
	}

	var notFoundErr *serviceerror.NotFound
	if errors.As(err, &notFoundErr) {
		return &errs.Error{Code: errs.NotFound, Message: "bill not found"}
	}
	return &errs.Error{Code: errs.Internal, Message: "workflow operation failed"}
}
