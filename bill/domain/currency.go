package domain

import (
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// Currency represents a supported ISO currency code for billing.
type Currency string

// Supported currency values for the Fees API.
const (
	USD Currency = "USD"
	GEL Currency = "GEL"
)

// maximum possible decimal places for supported currencies.
const maxDecimalPlaces = 2

var (
	// ErrUnsupportedCurrency indicates currency is not one of the supported currencies {USD, GEL}.
	ErrUnsupportedCurrency = errors.New("unsupported currency")

	// ErrAmountEmpty indicates the currency value passing in string is an empty string.
	ErrAmountEmpty = errors.New("currency value not set")

	// ErrAmountNegative indicates a negative value was passed for the amount.
	ErrAmountNegative = errors.New("negative value for currency")

	// ErrInvalidFormat indicates the amount passed as string has non-numeric characters('.' excluded).
	ErrInvalidFormat = errors.New("amount is not a valid decimal number")

	// ErrDecimalOverflow indicates the amount has too many decimal place unnatural for supported money types.
	ErrDecimalOverflow = errors.New("too many digits after decimal point")
)

// decimalPattern is used to validate if the passed amount is a decimal number.
var decimalPattern = regexp.MustCompile(`^[+-]?\d+(\.\d+)?$`)

// ParseCurrency checks if the passed string represents a supported currency value.
func ParseCurrency(ISOCode string) (Currency, error) {
	switch strings.ToUpper(ISOCode) {
	case "USD":
		return USD, nil
	case "GEL":
		return GEL, nil
	default:
		return "", fmt.Errorf("%w: %s", ErrUnsupportedCurrency, ISOCode)
	}
}

// ToMinorUnits converts a decimal amount in string to int64 representation in minor units.
// It does proper validation to ensure the passed amount is a valid decimal.
func ToMinorUnits(amount string) (int64, error) {
	if amount == "" {
		return 0, ErrAmountEmpty
	}

	amount = strings.TrimSpace(amount)

	if !decimalPattern.MatchString(amount) {
		return 0, fmt.Errorf("%w: %s", ErrInvalidFormat, amount)
	}

	if strings.HasPrefix(amount, "-") {
		return 0, fmt.Errorf("%w: %s", ErrAmountNegative, amount)
	}

	parts := strings.SplitN(amount, ".", 2)

	major, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid amount %s: %w", amount, err)
	}
	var minor int64
	if len(parts) == 2 {
		if len(parts[1]) > maxDecimalPlaces {
			return 0, fmt.Errorf("%w: %s", ErrDecimalOverflow, amount)
		}

		// pad 0 for correct value
		fraction := parts[1]
		fraction = fraction + strings.Repeat("0", 2-len(fraction))
		minor, err = strconv.ParseInt(fraction, 10, 64)
		if err != nil {
			return 0, fmt.Errorf("invalid amount %s: %w", amount, err)
		}
	}

	return major*100 + minor, nil
}

// FromMinorUnits converts an int64 number into a string that represents decimal amount
// It assumes minor >= 0.
// Minor value created by ToMinorUnits() will reject negative numbers.
// Negative numbers won't reach here.
func FromMinorUnits(minor int64) string {
	return fmt.Sprintf("%d.%02d", minor/100, minor%100)
}
