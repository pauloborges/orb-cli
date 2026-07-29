package mcpserver

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	orb "github.com/orbcorp/orb-go"
	"github.com/orbcorp/orb-go/option"
)

func TestTimeRangeAcceptsTimestampsAndDates(t *testing.T) {
	bounds, err := (&TimeRange{Gte: "2026-07-01", Lt: "2026-08-01T12:30:00Z"}).parse("created_at")
	if err != nil {
		t.Fatalf("parse() error = %v", err)
	}
	if bounds.gte == nil || bounds.gte.Format(dateLayout) != "2026-07-01" {
		t.Errorf("gte = %v, want 2026-07-01", bounds.gte)
	}
	if bounds.lt == nil || bounds.lt.Hour() != 12 {
		t.Errorf("lt = %v, want a timestamp at 12:30", bounds.lt)
	}
	if bounds.gt != nil || bounds.lte != nil {
		t.Error("parse() set a bound that the input left empty")
	}
}

func TestRangesRejectBadInput(t *testing.T) {
	if _, err := (&TimeRange{Gte: "last tuesday"}).parse("created_at"); err == nil {
		t.Error("parse() accepted a timestamp that is not RFC 3339")
	}
	if _, err := (&TimeRange{Gte: "2026-08-01", Lt: "2026-07-01"}).parse("created_at"); err == nil {
		t.Error("parse() accepted a lower bound after its upper bound")
	}
	if _, err := (&TimeRange{Gt: "2026-08-01", Lte: "2026-08-01"}).parse("created_at"); err == nil {
		t.Error("parse() accepted a range that can never match")
	}
	if _, err := (&DateRange{Gt: "2026-08-01", Lt: "2026-07-01"}).parse("due_date"); err == nil {
		t.Error("DateRange.parse() accepted a lower bound after its upper bound")
	}
	if _, err := (&TimeRange{Gte: "2026-08-01", Lte: "2026-08-01"}).parse("created_at"); err != nil {
		t.Errorf("parse() rejected two equal inclusive bounds: %v", err)
	}
	if err := (&AmountRange{Gt: "a lot"}).validate("amount"); err == nil {
		t.Error("validate() accepted an amount that is not a number")
	}
}

func TestDueDateWindowValidation(t *testing.T) {
	for _, window := range []string{"", "7d", "2m"} {
		if err := validateDueDateWindow(window); err != nil {
			t.Errorf("validateDueDateWindow(%q) error = %v", window, err)
		}
	}
	for _, window := range []string{"d", "7", "7y", "-1d", "sevend"} {
		if err := validateDueDateWindow(window); err == nil {
			t.Errorf("validateDueDateWindow(%q) accepted a bad window", window)
		}
	}
}

// TestFilterSchemasReachTheClient guards the embedded InvoiceFilters: the
// filters are useless if the schema does not show them to the model.
func TestFilterSchemasReachTheClient(t *testing.T) {
	ctx := context.Background()
	clientSession := connectTestClient(t, New(fakeReader{}))

	result, err := clientSession.ListTools(ctx, nil)
	if err != nil {
		t.Fatalf("ListTools() error = %v", err)
	}
	want := map[string][]string{
		"orb_list_customers":         {"created_at"},
		"orb_list_subscriptions":     {"created_at"},
		"orb_list_plans":             {"created_at"},
		"orb_list_invoices":          {"invoice_date", "due_date", "due_date_window", "amount", "is_recurring", "date_type"},
		"orb_list_invoice_summaries": {"invoice_date", "due_date", "due_date_window", "amount", "is_recurring", "date_type"},
	}
	for _, tool := range result.Tools {
		properties, ok := want[tool.Name]
		if !ok {
			continue
		}
		delete(want, tool.Name)
		schema := decodeSchemaProperties(t, tool.Name, tool.InputSchema)
		for _, property := range properties {
			if _, ok := schema[property]; !ok {
				t.Errorf("%s input schema is missing %s", tool.Name, property)
			}
		}
	}
	for name := range want {
		t.Errorf("tool %s was not registered", name)
	}
}

// TestUnsupportedBoundsFailLoudly covers the fields where Orb has no gte or
// lte. The schema must reject the bound, because a bound that this server drops
// in silence gives an answer that looks correct and is not.
func TestUnsupportedBoundsFailLoudly(t *testing.T) {
	ctx := context.Background()
	clientSession := connectTestClient(t, New(fakeReader{}))

	unsupported := []map[string]any{
		{"due_date": map[string]any{"gte": "2026-07-01"}},
		{"due_date": map[string]any{"lte": "2026-07-31"}},
		{"amount": map[string]any{"gte": "100"}},
		{"amount": map[string]any{"lte": "500"}},
	}
	for _, arguments := range unsupported {
		arguments["environment"] = "test"
		result, err := clientSession.CallTool(ctx, &mcp.CallToolParams{Name: "orb_list_invoices", Arguments: arguments})
		if err != nil {
			t.Fatalf("CallTool() error = %v", err)
		}
		if !result.IsError {
			t.Errorf("orb_list_invoices accepted an unsupported bound in %v", arguments)
		}
	}
}

// decodeSchemaProperties reads the properties of a tool input schema as the
// client receives it, which is raw JSON.
func decodeSchemaProperties(t *testing.T, tool string, schema any) map[string]any {
	t.Helper()
	encoded, err := json.Marshal(schema)
	if err != nil {
		t.Fatalf("could not encode the %s input schema: %v", tool, err)
	}
	var decoded struct {
		Properties map[string]any `json:"properties"`
	}
	if err := json.Unmarshal(encoded, &decoded); err != nil {
		t.Fatalf("could not decode the %s input schema: %v", tool, err)
	}
	return decoded.Properties
}

// TestFiltersReachOrbQueryString proves that each filter arrives in the query
// string with the bracket names and date formats that Orb expects.
func TestFiltersReachOrbQueryString(t *testing.T) {
	ctx := context.Background()
	isRecurring := true

	tests := []struct {
		name  string
		call  func(orbReader) error
		query map[string]string
	}{
		{
			name: "customers",
			call: func(reader orbReader) error {
				_, err := reader.ListCustomers(ctx, ListCustomersInput{
					PageInput: PageInput{Environment: EnvironmentTest},
					CreatedAt: &TimeRange{Gte: "2026-07-01", Lt: "2026-08-01T00:00:00Z"},
				})
				return err
			},
			query: map[string]string{
				"created_at[gte]": "2026-07-01T00:00:00Z",
				"created_at[lt]":  "2026-08-01T00:00:00Z",
			},
		},
		{
			name: "subscriptions",
			call: func(reader orbReader) error {
				_, err := reader.ListSubscriptions(ctx, ListSubscriptionsInput{
					PageInput: PageInput{Environment: EnvironmentTest},
					CreatedAt: &TimeRange{Gt: "2026-07-01", Lte: "2026-08-01"},
				})
				return err
			},
			query: map[string]string{
				"created_at[gt]":  "2026-07-01T00:00:00Z",
				"created_at[lte]": "2026-08-01T00:00:00Z",
			},
		},
		{
			name: "plans",
			call: func(reader orbReader) error {
				_, err := reader.ListPlans(ctx, ListPlansInput{
					PageInput: PageInput{Environment: EnvironmentTest},
					CreatedAt: &TimeRange{Gte: "2026-01-01"},
				})
				return err
			},
			query: map[string]string{"created_at[gte]": "2026-01-01T00:00:00Z"},
		},
		{
			name: "invoices",
			call: func(reader orbReader) error {
				_, err := reader.ListInvoices(ctx, ListInvoicesInput{
					PageInput: PageInput{Environment: EnvironmentTest},
					InvoiceFilters: InvoiceFilters{
						InvoiceDate:   &TimeRange{Gte: "2026-07-01", Lte: "2026-07-31"},
						DueDate:       &DateRange{Gt: "2026-07-05", Lt: "2026-07-20"},
						DueDateWindow: "7d",
						Amount:        &AmountRange{Gt: "100.00", Lt: "500.50"},
						IsRecurring:   &isRecurring,
						DateType:      "invoice_date",
					},
				})
				return err
			},
			query: map[string]string{
				"invoice_date[gte]": "2026-07-01T00:00:00Z",
				"invoice_date[lte]": "2026-07-31T00:00:00Z",
				"due_date[gt]":      "2026-07-05",
				"due_date[lt]":      "2026-07-20",
				"due_date_window":   "7d",
				"amount[gt]":        "100.00",
				"amount[lt]":        "500.50",
				"is_recurring":      "true",
				"date_type":         "invoice_date",
			},
		},
		{
			name: "invoice summaries",
			call: func(reader orbReader) error {
				_, err := reader.ListInvoiceSummaries(ctx, ListInvoiceSummariesInput{
					PageInput: PageInput{Environment: EnvironmentTest},
					InvoiceFilters: InvoiceFilters{
						InvoiceDate: &TimeRange{Gt: "2026-07-01T06:00:00Z"},
						DueDate:     &DateRange{Eq: "2026-07-31"},
						Amount:      &AmountRange{Eq: "42"},
						DateType:    "due_date",
					},
				})
				return err
			},
			query: map[string]string{
				"invoice_date[gt]": "2026-07-01T06:00:00Z",
				"due_date":         "2026-07-31",
				"amount":           "42",
				"date_type":        "due_date",
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var got url.Values
			reader := readerForTestServer(t, func(request *http.Request) {
				got = request.URL.Query()
			})
			if err := test.call(reader); err != nil {
				t.Fatalf("list call error = %v", err)
			}
			for key, want := range test.query {
				if got.Get(key) != want {
					t.Errorf("query %s = %q, want %q", key, got.Get(key), want)
				}
			}
		})
	}
}

// connectTestClient joins a client and the server over an in-memory transport.
func connectTestClient(t *testing.T, server *mcp.Server) *mcp.ClientSession {
	t.Helper()
	ctx := context.Background()
	clientTransport, serverTransport := mcp.NewInMemoryTransports()
	serverSession, err := server.Connect(ctx, serverTransport, nil)
	if err != nil {
		t.Fatalf("server.Connect() error = %v", err)
	}
	t.Cleanup(func() { serverSession.Close() })

	client := mcp.NewClient(&mcp.Implementation{Name: "test-client", Version: "0.0.0"}, nil)
	clientSession, err := client.Connect(ctx, clientTransport, nil)
	if err != nil {
		t.Fatalf("client.Connect() error = %v", err)
	}
	t.Cleanup(func() { clientSession.Close() })
	return clientSession
}

// readerForTestServer builds a reader whose test environment client talks to a
// local server, so a test can read the outgoing request.
func readerForTestServer(t *testing.T, record func(*http.Request)) orbReader {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		record(request)
		writer.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(writer).Encode(map[string]any{
			"data":                []any{},
			"pagination_metadata": map[string]any{"has_more": false, "next_cursor": nil},
		}); err != nil {
			t.Errorf("could not write the test response: %v", err)
		}
	}))
	t.Cleanup(server.Close)

	client := orb.NewClient(option.WithAPIKey("test-key"), option.WithBaseURL(server.URL+"/"))
	return orbReader{clients: map[Environment]*orb.Client{EnvironmentTest: client}}
}
