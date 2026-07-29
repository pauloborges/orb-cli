// Package mcpserver exposes read-only Orb resources as MCP tools.
package mcpserver

import (
	"context"
	"fmt"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	orb "github.com/orbcorp/orb-go"
	"github.com/orbcorp/orb-go/option"
)

const defaultPageSize int64 = 20

const (
	// LiveAPIKeyEnv contains the API key for Orb's live environment.
	LiveAPIKeyEnv = "ORB_LIVE_API_KEY"
	// TestAPIKeyEnv contains the API key for Orb's test environment.
	TestAPIKeyEnv = "ORB_TEST_API_KEY"
)

// Environment identifies an isolated Orb account.
type Environment string

const (
	EnvironmentLive Environment = "live"
	EnvironmentTest Environment = "test"
)

// Config provides the credentials required to query both Orb environments.
type Config struct {
	LiveAPIKey string
	TestAPIKey string
}

// Run serves the Orb MCP server over stdin and stdout until the client disconnects.
func Run(ctx context.Context, config Config) error {
	reader, err := newOrbReader(config)
	if err != nil {
		return err
	}
	server := New(reader)
	return server.Run(ctx, &mcp.StdioTransport{})
}

// New creates an MCP server backed by the supplied read-only Orb client.
func New(reader Reader) *mcp.Server {
	server := mcp.NewServer(&mcp.Implementation{Name: "orb-mcp", Version: "0.1.0"}, nil)
	handlers := toolHandlers{reader: reader}

	mcp.AddTool(server, readOnlyTool("orb_list_customers", "List Orb customers, ordered from most recently created. Supply one customer identifier to return that customer only. Filter by created_at when needed."), handlers.listCustomers)
	mcp.AddTool(server, readOnlyTool("orb_list_subscriptions", "List Orb subscriptions. Filter by customer, plan, subscription status, or created_at when needed."), handlers.listSubscriptions)
	mcp.AddTool(server, readOnlyTool("orb_list_plans", "List Orb plans. Filter by plan status or created_at when needed."), handlers.listPlans)
	mcp.AddTool(server, readOnlyTool("orb_list_invoices", "List Orb invoices. Filter by customer, subscription, invoice status, invoice_date, due_date, or amount when needed."), handlers.listInvoices)
	mcp.AddTool(server, readOnlyTool("orb_list_invoice_summaries", "List lightweight Orb invoice summaries. Use this instead of orb_list_invoices when line item details are not needed. It takes the same filters."), handlers.listInvoiceSummaries)
	mcp.AddTool(server, readOnlyTool("orb_get_customer", "Get one Orb customer by its Orb ID or external customer ID."), handlers.getCustomer)
	mcp.AddTool(server, readOnlyTool("orb_get_subscription", "Get one Orb subscription by its Orb ID."), handlers.getSubscription)
	mcp.AddTool(server, readOnlyTool("orb_get_plan", "Get one Orb plan by its Orb ID or external plan ID."), handlers.getPlan)
	mcp.AddTool(server, readOnlyTool("orb_get_invoice", "Get one Orb invoice by its Orb ID."), handlers.getInvoice)

	return server
}

func readOnlyTool(name, description string) *mcp.Tool {
	return &mcp.Tool{
		Name:        name,
		Description: description,
		Annotations: &mcp.ToolAnnotations{
			ReadOnlyHint:  true,
			OpenWorldHint: new(true),
		},
	}
}

// Reader is the read-only portion of the Orb API currently exposed by this server.
// Keeping it small makes additional API wrappers straightforward to test and add.
type Reader interface {
	ListCustomers(context.Context, ListCustomersInput) (ListOutput, error)
	ListSubscriptions(context.Context, ListSubscriptionsInput) (ListOutput, error)
	ListPlans(context.Context, ListPlansInput) (ListOutput, error)
	ListInvoices(context.Context, ListInvoicesInput) (ListOutput, error)
	ListInvoiceSummaries(context.Context, ListInvoiceSummariesInput) (ListOutput, error)
	GetCustomer(context.Context, GetCustomerInput) (GetOutput, error)
	GetSubscription(context.Context, GetInput) (GetOutput, error)
	GetPlan(context.Context, GetPlanInput) (GetOutput, error)
	GetInvoice(context.Context, GetInput) (GetOutput, error)
}

// PageInput is shared pagination input for list tools.
type PageInput struct {
	// Environment is required for every tool call. There is intentionally no default.
	Environment Environment `json:"environment" jsonschema:"Required Orb environment to query: live or test."`
	Cursor      string      `json:"cursor,omitempty" jsonschema:"Cursor returned by the previous list call."`
	Limit       int64       `json:"limit,omitempty" jsonschema:"Number of items to return, from 1 to 100. Defaults to 20."`
}

// ListCustomersInput filters a customer list. Orb does not support customer
// identifiers on its customer list endpoint, so an identifier returns one item.
type ListCustomersInput struct {
	PageInput
	CustomerID         string     `json:"customer_id,omitempty" jsonschema:"Orb customer ID. Do not set with external_customer_id."`
	ExternalCustomerID string     `json:"external_customer_id,omitempty" jsonschema:"External customer ID. Do not set with customer_id."`
	CreatedAt          *TimeRange `json:"created_at,omitempty" jsonschema:"Filter by when the customer was created."`
}

// ListSubscriptionsInput filters a subscription list.
type ListSubscriptionsInput struct {
	PageInput
	CustomerIDs         []string   `json:"customer_ids,omitempty" jsonschema:"Orb customer IDs to include."`
	ExternalCustomerIDs []string   `json:"external_customer_ids,omitempty" jsonschema:"External customer IDs to include."`
	PlanID              string     `json:"plan_id,omitempty" jsonschema:"Orb plan ID to include."`
	ExternalPlanID      string     `json:"external_plan_id,omitempty" jsonschema:"External plan ID to include."`
	Status              string     `json:"status,omitempty" jsonschema:"Subscription status: active, ended, or upcoming."`
	CreatedAt           *TimeRange `json:"created_at,omitempty" jsonschema:"Filter by when the subscription was created."`
}

// ListPlansInput filters a plan list.
type ListPlansInput struct {
	PageInput
	Status    string     `json:"status,omitempty" jsonschema:"Plan status: active, archived, or draft."`
	CreatedAt *TimeRange `json:"created_at,omitempty" jsonschema:"Filter by when the plan was created."`
}

// ListInvoicesInput filters an invoice list.
type ListInvoicesInput struct {
	PageInput
	InvoiceFilters
	CustomerID         string   `json:"customer_id,omitempty" jsonschema:"Orb customer ID to include."`
	ExternalCustomerID string   `json:"external_customer_id,omitempty" jsonschema:"External customer ID to include."`
	SubscriptionID     string   `json:"subscription_id,omitempty" jsonschema:"Orb subscription ID to include."`
	Statuses           []string `json:"statuses,omitempty" jsonschema:"Invoice statuses to include: draft, issued, paid, synced, or void."`
}

// ListInvoiceSummariesInput filters the lightweight invoice summary list.
type ListInvoiceSummariesInput struct {
	PageInput
	InvoiceFilters
	CustomerID         string `json:"customer_id,omitempty" jsonschema:"Orb customer ID to include."`
	ExternalCustomerID string `json:"external_customer_id,omitempty" jsonschema:"External customer ID to include."`
	SubscriptionID     string `json:"subscription_id,omitempty" jsonschema:"Orb subscription ID to include."`
	Status             string `json:"status,omitempty" jsonschema:"Invoice status: draft, issued, paid, synced, or void."`
}

// GetInput identifies one Orb resource by its Orb ID.
type GetInput struct {
	Environment Environment `json:"environment" jsonschema:"Required Orb environment to query: live or test."`
	ID          string      `json:"id" jsonschema:"Required Orb resource ID."`
}

// GetCustomerInput identifies one customer by one of its identifiers.
type GetCustomerInput struct {
	Environment        Environment `json:"environment" jsonschema:"Required Orb environment to query: live or test."`
	CustomerID         string      `json:"customer_id,omitempty" jsonschema:"Orb customer ID. Do not set with external_customer_id."`
	ExternalCustomerID string      `json:"external_customer_id,omitempty" jsonschema:"External customer ID. Do not set with customer_id."`
}

// GetPlanInput identifies one plan by one of its identifiers.
type GetPlanInput struct {
	Environment    Environment `json:"environment" jsonschema:"Required Orb environment to query: live or test."`
	PlanID         string      `json:"plan_id,omitempty" jsonschema:"Orb plan ID. Do not set with external_plan_id."`
	ExternalPlanID string      `json:"external_plan_id,omitempty" jsonschema:"External plan ID. Do not set with plan_id."`
}

// ListOutput is returned by every list tool. Data contains the API resource objects.
type ListOutput struct {
	Data       any        `json:"data" jsonschema:"The Orb resources returned for this page."`
	Pagination Pagination `json:"pagination" jsonschema:"Cursor pagination information."`
}

// GetOutput is returned by every single-resource tool.
type GetOutput struct {
	Data any `json:"data" jsonschema:"The Orb resource."`
}

// Pagination describes how to retrieve the next list page.
type Pagination struct {
	HasMore    bool   `json:"has_more" jsonschema:"Whether another page is available."`
	NextCursor string `json:"next_cursor,omitempty" jsonschema:"Cursor to pass to the next call, when has_more is true."`
}

type toolHandlers struct{ reader Reader }

func (h toolHandlers) listCustomers(ctx context.Context, _ *mcp.CallToolRequest, input ListCustomersInput) (*mcp.CallToolResult, ListOutput, error) {
	if err := validatePageInput(input.PageInput); err != nil {
		return nil, ListOutput{}, err
	}
	if err := validateIdentifiers("customer_id", "external_customer_id", input.CustomerID, input.ExternalCustomerID); err != nil {
		return nil, ListOutput{}, err
	}
	if _, err := input.CreatedAt.parse("created_at"); err != nil {
		return nil, ListOutput{}, err
	}
	output, err := h.reader.ListCustomers(ctx, input)
	return nil, output, wrapListError("customers", err)
}

func (h toolHandlers) listSubscriptions(ctx context.Context, _ *mcp.CallToolRequest, input ListSubscriptionsInput) (*mcp.CallToolResult, ListOutput, error) {
	if err := validatePageInput(input.PageInput); err != nil {
		return nil, ListOutput{}, err
	}
	if input.Status != "" && !validSubscriptionStatus(input.Status) {
		return nil, ListOutput{}, fmt.Errorf("status must be one of: active, ended, upcoming")
	}
	if _, err := input.CreatedAt.parse("created_at"); err != nil {
		return nil, ListOutput{}, err
	}
	output, err := h.reader.ListSubscriptions(ctx, input)
	return nil, output, wrapListError("subscriptions", err)
}

func (h toolHandlers) listPlans(ctx context.Context, _ *mcp.CallToolRequest, input ListPlansInput) (*mcp.CallToolResult, ListOutput, error) {
	if err := validatePageInput(input.PageInput); err != nil {
		return nil, ListOutput{}, err
	}
	if input.Status != "" && !validPlanStatus(input.Status) {
		return nil, ListOutput{}, fmt.Errorf("status must be one of: active, archived, draft")
	}
	if _, err := input.CreatedAt.parse("created_at"); err != nil {
		return nil, ListOutput{}, err
	}
	output, err := h.reader.ListPlans(ctx, input)
	return nil, output, wrapListError("plans", err)
}

func (h toolHandlers) listInvoices(ctx context.Context, _ *mcp.CallToolRequest, input ListInvoicesInput) (*mcp.CallToolResult, ListOutput, error) {
	if err := validatePageInput(input.PageInput); err != nil {
		return nil, ListOutput{}, err
	}
	for _, status := range input.Statuses {
		if !validInvoiceStatus(status) {
			return nil, ListOutput{}, fmt.Errorf("invoice statuses must be: draft, issued, paid, synced, or void")
		}
	}
	if err := validateInvoiceFilters(input.InvoiceFilters); err != nil {
		return nil, ListOutput{}, err
	}
	output, err := h.reader.ListInvoices(ctx, input)
	return nil, output, wrapListError("invoices", err)
}

func (h toolHandlers) listInvoiceSummaries(ctx context.Context, _ *mcp.CallToolRequest, input ListInvoiceSummariesInput) (*mcp.CallToolResult, ListOutput, error) {
	if err := validatePageInput(input.PageInput); err != nil {
		return nil, ListOutput{}, err
	}
	if input.Status != "" && !validInvoiceStatus(input.Status) {
		return nil, ListOutput{}, fmt.Errorf("status must be one of: draft, issued, paid, synced, void")
	}
	if err := validateInvoiceFilters(input.InvoiceFilters); err != nil {
		return nil, ListOutput{}, err
	}
	output, err := h.reader.ListInvoiceSummaries(ctx, input)
	return nil, output, wrapListError("invoice summaries", err)
}

func (h toolHandlers) getCustomer(ctx context.Context, _ *mcp.CallToolRequest, input GetCustomerInput) (*mcp.CallToolResult, GetOutput, error) {
	if !validEnvironment(input.Environment) {
		return nil, GetOutput{}, fmt.Errorf("environment must be one of: live, test")
	}
	if err := requireIdentifier("customer_id", "external_customer_id", input.CustomerID, input.ExternalCustomerID); err != nil {
		return nil, GetOutput{}, err
	}
	output, err := h.reader.GetCustomer(ctx, input)
	return nil, output, wrapGetError("customer", err)
}

func (h toolHandlers) getSubscription(ctx context.Context, _ *mcp.CallToolRequest, input GetInput) (*mcp.CallToolResult, GetOutput, error) {
	output, err := h.getByID(ctx, input, "subscription", h.reader.GetSubscription)
	return nil, output, err
}

func (h toolHandlers) getPlan(ctx context.Context, _ *mcp.CallToolRequest, input GetPlanInput) (*mcp.CallToolResult, GetOutput, error) {
	if !validEnvironment(input.Environment) {
		return nil, GetOutput{}, fmt.Errorf("environment must be one of: live, test")
	}
	if err := requireIdentifier("plan_id", "external_plan_id", input.PlanID, input.ExternalPlanID); err != nil {
		return nil, GetOutput{}, err
	}
	output, err := h.reader.GetPlan(ctx, input)
	return nil, output, wrapGetError("plan", err)
}

func (h toolHandlers) getInvoice(ctx context.Context, _ *mcp.CallToolRequest, input GetInput) (*mcp.CallToolResult, GetOutput, error) {
	output, err := h.getByID(ctx, input, "invoice", h.reader.GetInvoice)
	return nil, output, err
}

func (h toolHandlers) getByID(ctx context.Context, input GetInput, resource string, get func(context.Context, GetInput) (GetOutput, error)) (GetOutput, error) {
	if !validEnvironment(input.Environment) {
		return GetOutput{}, fmt.Errorf("environment must be one of: live, test")
	}
	if input.ID == "" {
		return GetOutput{}, fmt.Errorf("id is required")
	}
	output, err := get(ctx, input)
	return output, wrapGetError(resource, err)
}

func validatePageInput(input PageInput) error {
	if !validEnvironment(input.Environment) {
		return fmt.Errorf("environment must be one of: live, test")
	}
	_, err := pageSize(input.Limit)
	return err
}

// validateIdentifiers rejects two identifiers for one resource. Two empty
// identifiers are allowed, because a list call without one returns every
// resource.
func validateIdentifiers(idField, externalIDField, id, externalID string) error {
	if id != "" && externalID != "" {
		return fmt.Errorf("set %s or %s, not both", idField, externalIDField)
	}
	return nil
}

// requireIdentifier also rejects two empty identifiers, for a call that reads
// one resource.
func requireIdentifier(idField, externalIDField, id, externalID string) error {
	if err := validateIdentifiers(idField, externalIDField, id, externalID); err != nil {
		return err
	}
	if id == "" && externalID == "" {
		return fmt.Errorf("set %s or %s", idField, externalIDField)
	}
	return nil
}

func pageSize(limit int64) (int64, error) {
	if limit == 0 {
		return defaultPageSize, nil
	}
	if limit < 1 || limit > 100 {
		return 0, fmt.Errorf("limit must be between 1 and 100")
	}
	return limit, nil
}

// wrapListError names the failed action for a list tool. Pass a plural resource.
func wrapListError(resource string, err error) error {
	return wrapError("list", resource, err)
}

// wrapGetError names the failed action for a single-resource tool. Pass a
// singular resource.
func wrapGetError(resource string, err error) error {
	return wrapError("get", resource, err)
}

func wrapError(action, resource string, err error) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("could not %s Orb %s: %w", action, resource, err)
}

func validSubscriptionStatus(status string) bool {
	return status == "active" || status == "ended" || status == "upcoming"
}

func validPlanStatus(status string) bool {
	return status == "active" || status == "archived" || status == "draft"
}

func validInvoiceStatus(status string) bool {
	return status == "draft" || status == "issued" || status == "paid" || status == "synced" || status == "void"
}

func validEnvironment(environment Environment) bool {
	return environment == EnvironmentLive || environment == EnvironmentTest
}

func newOrbReader(config Config) (orbReader, error) {
	if config.LiveAPIKey == "" && config.TestAPIKey == "" {
		return orbReader{}, fmt.Errorf("%s and %s are required to run the MCP server", LiveAPIKeyEnv, TestAPIKeyEnv)
	}
	if config.LiveAPIKey == "" {
		return orbReader{}, fmt.Errorf("%s is required to run the MCP server", LiveAPIKeyEnv)
	}
	if config.TestAPIKey == "" {
		return orbReader{}, fmt.Errorf("%s is required to run the MCP server", TestAPIKeyEnv)
	}

	return orbReader{clients: map[Environment]*orb.Client{
		EnvironmentLive: orb.NewClient(option.WithAPIKey(config.LiveAPIKey)),
		EnvironmentTest: orb.NewClient(option.WithAPIKey(config.TestAPIKey)),
	}}, nil
}
