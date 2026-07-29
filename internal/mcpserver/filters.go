package mcpserver

import (
	"fmt"
	"strconv"
	"time"
)

// dateLayout is the date-only form accepted in addition to RFC 3339 timestamps.
const dateLayout = "2006-01-02"

const timeFormatHelp = "an RFC 3339 timestamp such as 2026-07-01T00:00:00Z, or a date such as 2026-07-01"

// TimeRange bounds a timestamp field on a list query. Set only the bounds that
// you need. Orb applies every bound that is set.
type TimeRange struct {
	Gt  string `json:"gt,omitempty" jsonschema:"Only include items after this time, exclusive. RFC 3339 timestamp or YYYY-MM-DD date."`
	Gte string `json:"gte,omitempty" jsonschema:"Only include items at or after this time. RFC 3339 timestamp or YYYY-MM-DD date."`
	Lt  string `json:"lt,omitempty" jsonschema:"Only include items before this time, exclusive. RFC 3339 timestamp or YYYY-MM-DD date."`
	Lte string `json:"lte,omitempty" jsonschema:"Only include items at or before this time. RFC 3339 timestamp or YYYY-MM-DD date."`
}

// DateRange bounds a date field on a list query. Orb has no gte or lte here.
type DateRange struct {
	Eq string `json:"eq,omitempty" jsonschema:"Only include items on this exact date. YYYY-MM-DD."`
	Gt string `json:"gt,omitempty" jsonschema:"Only include items after this date, exclusive. YYYY-MM-DD."`
	Lt string `json:"lt,omitempty" jsonschema:"Only include items before this date, exclusive. YYYY-MM-DD."`
}

// AmountRange bounds an invoice total. Orb expects decimal strings, so the
// values stay as strings to keep full precision. Orb has no gte or lte here.
type AmountRange struct {
	Eq string `json:"eq,omitempty" jsonschema:"Only include invoices with this exact total, as a decimal string."`
	Gt string `json:"gt,omitempty" jsonschema:"Only include invoices above this total, exclusive, as a decimal string."`
	Lt string `json:"lt,omitempty" jsonschema:"Only include invoices below this total, exclusive, as a decimal string."`
}

// timeBounds holds the parsed form of a TimeRange. A nil field is not set.
type timeBounds struct{ gt, gte, lt, lte *time.Time }

// dateBounds holds the parsed form of a DateRange. A nil field is not set.
type dateBounds struct{ eq, gt, lt *time.Time }

// parse converts the range into times. The field name goes into error messages
// so that the caller knows which input was wrong.
func (r *TimeRange) parse(field string) (timeBounds, error) {
	var bounds timeBounds
	if r == nil {
		return bounds, nil
	}
	targets := []struct {
		name   string
		value  string
		target **time.Time
	}{
		{"gt", r.Gt, &bounds.gt},
		{"gte", r.Gte, &bounds.gte},
		{"lt", r.Lt, &bounds.lt},
		{"lte", r.Lte, &bounds.lte},
	}
	for _, target := range targets {
		parsed, err := parseOptionalTime(field+"."+target.name, target.value)
		if err != nil {
			return timeBounds{}, err
		}
		*target.target = parsed
	}
	lower, lowerInclusive := bounds.gt, false
	if lower == nil {
		lower, lowerInclusive = bounds.gte, true
	}
	upper, upperInclusive := bounds.lt, false
	if upper == nil {
		upper, upperInclusive = bounds.lte, true
	}
	if err := checkBoundOrder(field, lower, upper, lowerInclusive && upperInclusive); err != nil {
		return timeBounds{}, err
	}
	return bounds, nil
}

// parse converts the range into times. The field name goes into error messages
// so that the caller knows which input was wrong.
func (r *DateRange) parse(field string) (dateBounds, error) {
	var bounds dateBounds
	if r == nil {
		return bounds, nil
	}
	targets := []struct {
		name   string
		value  string
		target **time.Time
	}{
		{"eq", r.Eq, &bounds.eq},
		{"gt", r.Gt, &bounds.gt},
		{"lt", r.Lt, &bounds.lt},
	}
	for _, target := range targets {
		parsed, err := parseOptionalTime(field+"."+target.name, target.value)
		if err != nil {
			return dateBounds{}, err
		}
		*target.target = parsed
	}
	if err := checkBoundOrder(field, bounds.gt, bounds.lt, false); err != nil {
		return dateBounds{}, err
	}
	return bounds, nil
}

// validate makes sure that each amount is a decimal number.
func (r *AmountRange) validate(field string) error {
	if r == nil {
		return nil
	}
	values := []struct {
		name  string
		value string
	}{
		{"eq", r.Eq},
		{"gt", r.Gt},
		{"lt", r.Lt},
	}
	for _, value := range values {
		if value.value == "" {
			continue
		}
		if _, err := strconv.ParseFloat(value.value, 64); err != nil {
			return fmt.Errorf("%s.%s must be a decimal number such as 100.00", field, value.name)
		}
	}
	return nil
}

// parseOptionalTime returns nil when the value is empty.
func parseOptionalTime(field, value string) (*time.Time, error) {
	if value == "" {
		return nil, nil
	}
	if parsed, err := time.Parse(time.RFC3339, value); err == nil {
		return &parsed, nil
	}
	if parsed, err := time.Parse(dateLayout, value); err == nil {
		return &parsed, nil
	}
	return nil, fmt.Errorf("%s must be %s", field, timeFormatHelp)
}

// checkBoundOrder rejects a range that can never match, because an empty result
// looks the same as a correct query with no data. Two inclusive bounds can be
// equal, because that range still matches one point in time.
func checkBoundOrder(field string, lower, upper *time.Time, bothInclusive bool) error {
	if lower == nil || upper == nil {
		return nil
	}
	if lower.Before(*upper) || (bothInclusive && lower.Equal(*upper)) {
		return nil
	}
	return fmt.Errorf("%s lower bound must be before its upper bound", field)
}

// validateDueDateWindow checks Orb's relative window form, such as 7d or 2m.
func validateDueDateWindow(window string) error {
	if window == "" {
		return nil
	}
	unit := window[len(window)-1:]
	count := window[:len(window)-1]
	if (unit != "d" && unit != "m") || count == "" {
		return fmt.Errorf("due_date_window must be a number followed by d for days or m for months, such as 7d")
	}
	if _, err := strconv.ParseUint(count, 10, 32); err != nil {
		return fmt.Errorf("due_date_window must be a number followed by d for days or m for months, such as 7d")
	}
	return nil
}

// validateDateType checks which date field Orb filters and sorts invoices by.
func validateDateType(dateType string) error {
	if dateType == "" || dateType == "due_date" || dateType == "invoice_date" {
		return nil
	}
	return fmt.Errorf("date_type must be one of: due_date, invoice_date")
}

// InvoiceFilters is shared by the invoice list and invoice summary tools.
type InvoiceFilters struct {
	InvoiceDate   *TimeRange   `json:"invoice_date,omitempty" jsonschema:"Filter by the date Orb issued the invoice."`
	DueDate       *DateRange   `json:"due_date,omitempty" jsonschema:"Filter by the invoice due date."`
	DueDateWindow string       `json:"due_date_window,omitempty" jsonschema:"Filter by a due date window in the past, as a number followed by d for days or m for months, such as 7d."`
	Amount        *AmountRange `json:"amount,omitempty" jsonschema:"Filter by the invoice total."`
	IsRecurring   *bool        `json:"is_recurring,omitempty" jsonschema:"Set true for invoices from a recurring subscription, or false for one-off invoices."`
	DateType      string       `json:"date_type,omitempty" jsonschema:"Which date Orb filters and sorts by: due_date or invoice_date."`
}

// validateInvoiceFilters is a function rather than a method, so that it does not
// promote onto the input types that embed InvoiceFilters.
func validateInvoiceFilters(filters InvoiceFilters) error {
	if _, err := filters.InvoiceDate.parse("invoice_date"); err != nil {
		return err
	}
	if _, err := filters.DueDate.parse("due_date"); err != nil {
		return err
	}
	if err := validateDueDateWindow(filters.DueDateWindow); err != nil {
		return err
	}
	if err := filters.Amount.validate("amount"); err != nil {
		return err
	}
	return validateDateType(filters.DateType)
}
