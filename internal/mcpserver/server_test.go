package mcpserver

import (
	"context"
	"errors"
	"net/http"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type fakeReader struct {
	output ListOutput
	err    error
}

func (f fakeReader) ListCustomers(context.Context, ListCustomersInput) (ListOutput, error) {
	return f.output, f.err
}
func (f fakeReader) ListSubscriptions(context.Context, ListSubscriptionsInput) (ListOutput, error) {
	return f.output, f.err
}
func (f fakeReader) ListPlans(context.Context, ListPlansInput) (ListOutput, error) {
	return f.output, f.err
}
func (f fakeReader) ListInvoices(context.Context, ListInvoicesInput) (ListOutput, error) {
	return f.output, f.err
}
func (f fakeReader) ListInvoiceSummaries(context.Context, ListInvoiceSummariesInput) (ListOutput, error) {
	return f.output, f.err
}
func (f fakeReader) GetCustomer(context.Context, GetCustomerInput) (GetOutput, error) {
	return GetOutput{Data: f.output.Data}, f.err
}
func (f fakeReader) GetSubscription(context.Context, GetInput) (GetOutput, error) {
	return GetOutput{Data: f.output.Data}, f.err
}
func (f fakeReader) GetPlan(context.Context, GetPlanInput) (GetOutput, error) {
	return GetOutput{Data: f.output.Data}, f.err
}
func (f fakeReader) GetInvoice(context.Context, GetInput) (GetOutput, error) {
	return GetOutput{Data: f.output.Data}, f.err
}

func TestListCustomersReturnsReaderResult(t *testing.T) {
	want := ListOutput{Data: []string{"customer_1"}, Pagination: Pagination{HasMore: true, NextCursor: "next"}}
	handler := toolHandlers{reader: fakeReader{output: want}}

	_, got, err := handler.listCustomers(context.Background(), nil, ListCustomersInput{PageInput: PageInput{Environment: EnvironmentTest, Limit: 20}})
	if err != nil {
		t.Fatalf("listCustomers() error = %v", err)
	}
	if got.Pagination != want.Pagination {
		t.Fatalf("pagination = %#v, want %#v", got.Pagination, want.Pagination)
	}
}

func TestListToolsRejectInvalidInputBeforeCallingOrb(t *testing.T) {
	handler := toolHandlers{reader: fakeReader{err: errors.New("should not be called")}}

	if _, _, err := handler.listPlans(context.Background(), nil, ListPlansInput{PageInput: PageInput{Environment: EnvironmentTest, Limit: 101}}); err == nil {
		t.Fatal("listPlans() succeeded with an invalid limit")
	}
	if _, _, err := handler.listSubscriptions(context.Background(), nil, ListSubscriptionsInput{PageInput: PageInput{Environment: EnvironmentTest}, Status: "paused"}); err == nil {
		t.Fatal("listSubscriptions() succeeded with an invalid status")
	}
	if _, _, err := handler.listInvoices(context.Background(), nil, ListInvoicesInput{PageInput: PageInput{Environment: EnvironmentTest}, Statuses: []string{"pending"}}); err == nil {
		t.Fatal("listInvoices() succeeded with an invalid status")
	}
	if _, _, err := handler.listCustomers(context.Background(), nil, ListCustomersInput{PageInput: PageInput{Environment: EnvironmentTest}, CreatedAt: &TimeRange{Gte: "July"}}); err == nil {
		t.Fatal("listCustomers() succeeded with an invalid created_at")
	}
	if _, _, err := handler.listInvoices(context.Background(), nil, ListInvoicesInput{PageInput: PageInput{Environment: EnvironmentTest}, InvoiceFilters: InvoiceFilters{DateType: "paid_at"}}); err == nil {
		t.Fatal("listInvoices() succeeded with an invalid date_type")
	}
}

func TestListToolsRequireAnExplicitEnvironment(t *testing.T) {
	handler := toolHandlers{reader: fakeReader{err: errors.New("should not be called")}}

	if _, _, err := handler.listCustomers(context.Background(), nil, ListCustomersInput{}); err == nil {
		t.Fatal("listCustomers() succeeded without an environment")
	}
	if _, _, err := handler.listCustomers(context.Background(), nil, ListCustomersInput{PageInput: PageInput{Environment: "preview"}}); err == nil {
		t.Fatal("listCustomers() succeeded with an unknown environment")
	}
}

func TestGetCustomerRequiresOneIdentifier(t *testing.T) {
	handler := toolHandlers{reader: fakeReader{err: errors.New("should not be called")}}

	if _, _, err := handler.getCustomer(context.Background(), nil, GetCustomerInput{Environment: EnvironmentTest}); err == nil {
		t.Fatal("getCustomer() succeeded without an identifier")
	}
	if _, _, err := handler.getCustomer(context.Background(), nil, GetCustomerInput{Environment: EnvironmentTest, CustomerID: "customer_1", ExternalCustomerID: "external_1"}); err == nil {
		t.Fatal("getCustomer() succeeded with two identifiers")
	}
}

func TestGetPlanRequiresOneIdentifier(t *testing.T) {
	handler := toolHandlers{reader: fakeReader{err: errors.New("should not be called")}}

	if _, _, err := handler.getPlan(context.Background(), nil, GetPlanInput{Environment: EnvironmentTest}); err == nil {
		t.Fatal("getPlan() succeeded without an identifier")
	}
	if _, _, err := handler.getPlan(context.Background(), nil, GetPlanInput{Environment: EnvironmentTest, PlanID: "plan_1", ExternalPlanID: "external_1"}); err == nil {
		t.Fatal("getPlan() succeeded with two identifiers")
	}
}

// TestGetPlanChoosesEndpointByIdentifier proves that an external plan ID goes to
// Orb's external plan ID endpoint, not to the plan ID endpoint.
func TestGetPlanChoosesEndpointByIdentifier(t *testing.T) {
	ctx := context.Background()
	tests := []struct {
		name  string
		input GetPlanInput
		path  string
	}{
		{
			name:  "Orb ID",
			input: GetPlanInput{Environment: EnvironmentTest, PlanID: "plan_1"},
			path:  "/plans/plan_1",
		},
		{
			name:  "external ID",
			input: GetPlanInput{Environment: EnvironmentTest, ExternalPlanID: "external_1"},
			path:  "/plans/external_plan_id/external_1",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var got string
			reader := readerForTestServer(t, func(request *http.Request) {
				got = request.URL.Path
			})
			if _, err := reader.GetPlan(ctx, test.input); err != nil {
				t.Fatalf("GetPlan() error = %v", err)
			}
			if got != test.path {
				t.Errorf("path = %q, want %q", got, test.path)
			}
		})
	}
}

func TestGetResourceRequiresID(t *testing.T) {
	handler := toolHandlers{reader: fakeReader{err: errors.New("should not be called")}}

	if _, _, err := handler.getInvoice(context.Background(), nil, GetInput{Environment: EnvironmentTest}); err == nil {
		t.Fatal("getInvoice() succeeded without an ID")
	}
}

func TestNewOrbReaderRequiresBothEnvironmentKeys(t *testing.T) {
	if _, err := newOrbReader(Config{}); err == nil {
		t.Fatal("newOrbReader() succeeded without API keys")
	}
	if _, err := newOrbReader(Config{LiveAPIKey: "live-key"}); err == nil {
		t.Fatal("newOrbReader() succeeded without a test API key")
	}
	if _, err := newOrbReader(Config{TestAPIKey: "test-key"}); err == nil {
		t.Fatal("newOrbReader() succeeded without a live API key")
	}
}

func TestServerRegistersReadOnlyListTools(t *testing.T) {
	ctx := context.Background()
	server := New(fakeReader{})
	clientTransport, serverTransport := mcp.NewInMemoryTransports()
	serverSession, err := server.Connect(ctx, serverTransport, nil)
	if err != nil {
		t.Fatalf("server.Connect() error = %v", err)
	}
	defer serverSession.Close()

	client := mcp.NewClient(&mcp.Implementation{Name: "test-client", Version: "0.0.0"}, nil)
	clientSession, err := client.Connect(ctx, clientTransport, nil)
	if err != nil {
		t.Fatalf("client.Connect() error = %v", err)
	}
	defer clientSession.Close()

	result, err := clientSession.ListTools(ctx, nil)
	if err != nil {
		t.Fatalf("ListTools() error = %v", err)
	}
	want := map[string]bool{
		"orb_list_customers":         false,
		"orb_list_subscriptions":     false,
		"orb_list_plans":             false,
		"orb_list_invoices":          false,
		"orb_list_invoice_summaries": false,
		"orb_get_customer":           false,
		"orb_get_subscription":       false,
		"orb_get_plan":               false,
		"orb_get_invoice":            false,
	}
	for _, tool := range result.Tools {
		if _, ok := want[tool.Name]; ok {
			if tool.Annotations == nil || !tool.Annotations.ReadOnlyHint {
				t.Errorf("%s is not marked read-only", tool.Name)
			}
			want[tool.Name] = true
		}
	}
	for name, found := range want {
		if !found {
			t.Errorf("tool %s was not registered", name)
		}
	}
}
