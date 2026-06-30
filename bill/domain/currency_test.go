package domain

import (
	"errors"
	"testing"
)

func TestCheckSupported(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    Currency
		wantErr error
	}{
		{name: "accept dollar", input: "USD", want: Currency(USD), wantErr: nil},
		{name: "accept lari", input: "GEL", want: Currency(GEL), wantErr: nil},
		{name: "reject unsupported", input: "INR", want: "", wantErr: ErrUnsupportedCurrency},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseCurrency(tt.input)
			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("got err %v, want %v", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.want {
				t.Errorf("CheckSupported(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestToMinorUnits(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    int64
		wantErr error
	}{
		{name: "whole number", input: "23", want: 2300, wantErr: nil},
		{name: "decimal number", input: "23.32", want: 2332, wantErr: nil},
		{name: "trailing and leading spaces", input: " 23.32  ", want: 2332, wantErr: nil},
		{name: "negative number", input: "-23.32", want: 0, wantErr: ErrCurrencyNegative},
		{name: "mixed string", input: "-A23f.$%32", want: 0, wantErr: ErrInvalidFormat},
		{name: "empty string", input: "", want: 0, wantErr: ErrCurrencyEmpty},
		{name: "long decimal", input: "23.3223", want: 0, wantErr: ErrDecimalOverflow},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ToMinorUnits(tt.input)
			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("got err %v want err %v", err, tt.wantErr)
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error %v", err)
			}

			if got != tt.want {
				t.Errorf("ToMinorUnits(%q) = %d, want %d", tt.input, got, tt.want)
			}
		})
	}
}

func TestFromMinorUnits(t *testing.T) {
	tests := []struct {
		name  string
		input int64
		want  string
	}{
		{name: "four digits", input: 2332, want: "23.32"},
		{name: "three digits", input: 100, want: "1.00"},
		{name: "one digit", input: 1, want: "0.01"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := FromMinorUnits(tt.input)
			if got != tt.want {
				t.Fatalf("got %q want %q", got, tt.want)
				return
			}
		})
	}
}
