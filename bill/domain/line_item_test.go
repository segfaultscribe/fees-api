package domain

import (
	"errors"
	"testing"
)

func TestNewLineItem(t *testing.T) {
	tests := []struct {
		name             string
		inputLineID      string
		inputDescription string
		inputAmount      int64
		inputCurrency    Currency
		wantDescription  string
		wantErr          error
	}{
		{name: "valid line item", inputLineID: "test-id", inputDescription: "cheeseburger", inputAmount: 1200, inputCurrency: Currency(USD), wantDescription: "cheeseburger", wantErr: nil},
		{name: "description with spaces", inputLineID: "test-id", inputDescription: " cheeseburger  ", inputAmount: 1200, inputCurrency: Currency(USD), wantDescription: "cheeseburger", wantErr: nil},
		{name: "empty description", inputLineID: "test-id", inputDescription: "", inputAmount: 1200, inputCurrency: USD, wantDescription: "", wantErr: ErrDescriptionEmpty},
		{name: "empty line_id", inputLineID: "", inputDescription: "cheeseburger", inputAmount: 1200, inputCurrency: USD, wantDescription: "", wantErr: ErrLineIDEmpty},
		{name: "negative amount", inputLineID: "test-id", inputDescription: "item", inputAmount: -100, inputCurrency: USD, wantDescription: "", wantErr: ErrAmountNegative},
		{name: "zero amount", inputLineID: "test-id", inputDescription: "gift", inputAmount: 0, inputCurrency: USD, wantDescription: "gift", wantErr: nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NewLineItem(tt.inputLineID, tt.inputDescription, tt.inputAmount, tt.inputCurrency)
			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("got err %v want err %v", err, tt.wantErr)
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error %v", err)
			}

			if got.Description != tt.wantDescription {
				t.Errorf("Description = %q, want %q", got.Description, tt.inputDescription)
			}
			if got.Amount != tt.inputAmount {
				t.Errorf("Amount = %d, want %d", got.Amount, tt.inputAmount)
			}
			if got.Currency != tt.inputCurrency {
				t.Errorf("Currency = %v, want %v", got.Currency, tt.inputCurrency)
			}
			if got.LineID != tt.inputLineID {
				t.Errorf("LineID = %q, want %q", got.LineID, tt.inputLineID)
			}
		})
	}
}
