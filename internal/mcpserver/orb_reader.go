package mcpserver

import (
	"context"
	"fmt"

	orb "github.com/orbcorp/orb-go"
)

type orbReader struct{ clients map[Environment]*orb.Client }

func (r orbReader) ListCustomers(ctx context.Context, input ListCustomersInput) (ListOutput, error) {
	client, err := r.clientFor(input.Environment)
	if err != nil {
		return ListOutput{}, err
	}
	if input.CustomerID != "" || input.ExternalCustomerID != "" {
		customer, err := r.GetCustomer(ctx, GetCustomerInput{
			Environment:        input.Environment,
			CustomerID:         input.CustomerID,
			ExternalCustomerID: input.ExternalCustomerID,
		})
		if err != nil {
			return ListOutput{}, err
		}
		return listOutput([]any{customer.Data}, false, ""), nil
	}
	limit, err := pageSize(input.Limit)
	if err != nil {
		return ListOutput{}, err
	}
	createdAt, err := input.CreatedAt.parse("created_at")
	if err != nil {
		return ListOutput{}, err
	}
	params := orb.CustomerListParams{Limit: orb.F(limit)}
	if input.Cursor != "" {
		params.Cursor = orb.F(input.Cursor)
	}
	if createdAt.gt != nil {
		params.CreatedAtGt = orb.F(*createdAt.gt)
	}
	if createdAt.gte != nil {
		params.CreatedAtGte = orb.F(*createdAt.gte)
	}
	if createdAt.lt != nil {
		params.CreatedAtLt = orb.F(*createdAt.lt)
	}
	if createdAt.lte != nil {
		params.CreatedAtLte = orb.F(*createdAt.lte)
	}
	page, err := client.Customers.List(ctx, params)
	if err != nil {
		return ListOutput{}, err
	}
	return listOutput(page.Data, page.PaginationMetadata.HasMore, page.PaginationMetadata.NextCursor), nil
}

func (r orbReader) ListSubscriptions(ctx context.Context, input ListSubscriptionsInput) (ListOutput, error) {
	client, err := r.clientFor(input.Environment)
	if err != nil {
		return ListOutput{}, err
	}
	limit, err := pageSize(input.Limit)
	if err != nil {
		return ListOutput{}, err
	}
	createdAt, err := input.CreatedAt.parse("created_at")
	if err != nil {
		return ListOutput{}, err
	}
	params := orb.SubscriptionListParams{Limit: orb.F(limit)}
	if input.Cursor != "" {
		params.Cursor = orb.F(input.Cursor)
	}
	if createdAt.gt != nil {
		params.CreatedAtGt = orb.F(*createdAt.gt)
	}
	if createdAt.gte != nil {
		params.CreatedAtGte = orb.F(*createdAt.gte)
	}
	if createdAt.lt != nil {
		params.CreatedAtLt = orb.F(*createdAt.lt)
	}
	if createdAt.lte != nil {
		params.CreatedAtLte = orb.F(*createdAt.lte)
	}
	if len(input.CustomerIDs) > 0 {
		params.CustomerID = orb.F(input.CustomerIDs)
	}
	if len(input.ExternalCustomerIDs) > 0 {
		params.ExternalCustomerID = orb.F(input.ExternalCustomerIDs)
	}
	if input.PlanID != "" {
		params.PlanID = orb.F(input.PlanID)
	}
	if input.ExternalPlanID != "" {
		params.ExternalPlanID = orb.F(input.ExternalPlanID)
	}
	if input.Status != "" {
		params.Status = orb.F(orb.SubscriptionListParamsStatus(input.Status))
	}
	page, err := client.Subscriptions.List(ctx, params)
	if err != nil {
		return ListOutput{}, err
	}
	return listOutput(page.Data, page.PaginationMetadata.HasMore, page.PaginationMetadata.NextCursor), nil
}

func (r orbReader) ListPlans(ctx context.Context, input ListPlansInput) (ListOutput, error) {
	client, err := r.clientFor(input.Environment)
	if err != nil {
		return ListOutput{}, err
	}
	limit, err := pageSize(input.Limit)
	if err != nil {
		return ListOutput{}, err
	}
	createdAt, err := input.CreatedAt.parse("created_at")
	if err != nil {
		return ListOutput{}, err
	}
	params := orb.PlanListParams{Limit: orb.F(limit)}
	if input.Cursor != "" {
		params.Cursor = orb.F(input.Cursor)
	}
	if input.Status != "" {
		params.Status = orb.F(orb.PlanListParamsStatus(input.Status))
	}
	if createdAt.gt != nil {
		params.CreatedAtGt = orb.F(*createdAt.gt)
	}
	if createdAt.gte != nil {
		params.CreatedAtGte = orb.F(*createdAt.gte)
	}
	if createdAt.lt != nil {
		params.CreatedAtLt = orb.F(*createdAt.lt)
	}
	if createdAt.lte != nil {
		params.CreatedAtLte = orb.F(*createdAt.lte)
	}
	page, err := client.Plans.List(ctx, params)
	if err != nil {
		return ListOutput{}, err
	}
	return listOutput(page.Data, page.PaginationMetadata.HasMore, page.PaginationMetadata.NextCursor), nil
}

func (r orbReader) ListInvoices(ctx context.Context, input ListInvoicesInput) (ListOutput, error) {
	client, err := r.clientFor(input.Environment)
	if err != nil {
		return ListOutput{}, err
	}
	limit, err := pageSize(input.Limit)
	if err != nil {
		return ListOutput{}, err
	}
	invoiceDate, err := input.InvoiceDate.parse("invoice_date")
	if err != nil {
		return ListOutput{}, err
	}
	dueDate, err := input.DueDate.parse("due_date")
	if err != nil {
		return ListOutput{}, err
	}
	params := orb.InvoiceListParams{Limit: orb.F(limit)}
	if input.Cursor != "" {
		params.Cursor = orb.F(input.Cursor)
	}
	if input.CustomerID != "" {
		params.CustomerID = orb.F(input.CustomerID)
	}
	if input.ExternalCustomerID != "" {
		params.ExternalCustomerID = orb.F(input.ExternalCustomerID)
	}
	if input.SubscriptionID != "" {
		params.SubscriptionID = orb.F(input.SubscriptionID)
	}
	if invoiceDate.gt != nil {
		params.InvoiceDateGt = orb.F(*invoiceDate.gt)
	}
	if invoiceDate.gte != nil {
		params.InvoiceDateGte = orb.F(*invoiceDate.gte)
	}
	if invoiceDate.lt != nil {
		params.InvoiceDateLt = orb.F(*invoiceDate.lt)
	}
	if invoiceDate.lte != nil {
		params.InvoiceDateLte = orb.F(*invoiceDate.lte)
	}
	if dueDate.eq != nil {
		params.DueDate = orb.F(*dueDate.eq)
	}
	if dueDate.gt != nil {
		params.DueDateGt = orb.F(*dueDate.gt)
	}
	if dueDate.lt != nil {
		params.DueDateLt = orb.F(*dueDate.lt)
	}
	if input.DueDateWindow != "" {
		params.DueDateWindow = orb.F(input.DueDateWindow)
	}
	if input.Amount != nil {
		if input.Amount.Eq != "" {
			params.Amount = orb.F(input.Amount.Eq)
		}
		if input.Amount.Gt != "" {
			params.AmountGt = orb.F(input.Amount.Gt)
		}
		if input.Amount.Lt != "" {
			params.AmountLt = orb.F(input.Amount.Lt)
		}
	}
	if input.IsRecurring != nil {
		params.IsRecurring = orb.F(*input.IsRecurring)
	}
	if input.DateType != "" {
		params.DateType = orb.F(orb.InvoiceListParamsDateType(input.DateType))
	}
	if len(input.Statuses) > 0 {
		statuses := make([]orb.InvoiceListParamsStatus, len(input.Statuses))
		for i, status := range input.Statuses {
			statuses[i] = orb.InvoiceListParamsStatus(status)
		}
		params.Status = orb.F(statuses)
	}
	page, err := client.Invoices.List(ctx, params)
	if err != nil {
		return ListOutput{}, err
	}
	return listOutput(page.Data, page.PaginationMetadata.HasMore, page.PaginationMetadata.NextCursor), nil
}

func (r orbReader) ListInvoiceSummaries(ctx context.Context, input ListInvoiceSummariesInput) (ListOutput, error) {
	client, err := r.clientFor(input.Environment)
	if err != nil {
		return ListOutput{}, err
	}
	limit, err := pageSize(input.Limit)
	if err != nil {
		return ListOutput{}, err
	}
	invoiceDate, err := input.InvoiceDate.parse("invoice_date")
	if err != nil {
		return ListOutput{}, err
	}
	dueDate, err := input.DueDate.parse("due_date")
	if err != nil {
		return ListOutput{}, err
	}
	params := orb.InvoiceListSummaryParams{Limit: orb.F(limit)}
	if input.Cursor != "" {
		params.Cursor = orb.F(input.Cursor)
	}
	if input.CustomerID != "" {
		params.CustomerID = orb.F(input.CustomerID)
	}
	if input.ExternalCustomerID != "" {
		params.ExternalCustomerID = orb.F(input.ExternalCustomerID)
	}
	if input.SubscriptionID != "" {
		params.SubscriptionID = orb.F(input.SubscriptionID)
	}
	if input.Status != "" {
		params.Status = orb.F(orb.InvoiceListSummaryParamsStatus(input.Status))
	}
	if invoiceDate.gt != nil {
		params.InvoiceDateGt = orb.F(*invoiceDate.gt)
	}
	if invoiceDate.gte != nil {
		params.InvoiceDateGte = orb.F(*invoiceDate.gte)
	}
	if invoiceDate.lt != nil {
		params.InvoiceDateLt = orb.F(*invoiceDate.lt)
	}
	if invoiceDate.lte != nil {
		params.InvoiceDateLte = orb.F(*invoiceDate.lte)
	}
	if dueDate.eq != nil {
		params.DueDate = orb.F(*dueDate.eq)
	}
	if dueDate.gt != nil {
		params.DueDateGt = orb.F(*dueDate.gt)
	}
	if dueDate.lt != nil {
		params.DueDateLt = orb.F(*dueDate.lt)
	}
	if input.DueDateWindow != "" {
		params.DueDateWindow = orb.F(input.DueDateWindow)
	}
	if input.Amount != nil {
		if input.Amount.Eq != "" {
			params.Amount = orb.F(input.Amount.Eq)
		}
		if input.Amount.Gt != "" {
			params.AmountGt = orb.F(input.Amount.Gt)
		}
		if input.Amount.Lt != "" {
			params.AmountLt = orb.F(input.Amount.Lt)
		}
	}
	if input.IsRecurring != nil {
		params.IsRecurring = orb.F(*input.IsRecurring)
	}
	if input.DateType != "" {
		params.DateType = orb.F(orb.InvoiceListSummaryParamsDateType(input.DateType))
	}
	page, err := client.Invoices.ListSummary(ctx, params)
	if err != nil {
		return ListOutput{}, err
	}
	return listOutput(page.Data, page.PaginationMetadata.HasMore, page.PaginationMetadata.NextCursor), nil
}

func (r orbReader) GetCustomer(ctx context.Context, input GetCustomerInput) (GetOutput, error) {
	client, err := r.clientFor(input.Environment)
	if err != nil {
		return GetOutput{}, err
	}
	if input.CustomerID != "" {
		customer, err := client.Customers.Fetch(ctx, input.CustomerID)
		return GetOutput{Data: customer}, err
	}
	customer, err := client.Customers.FetchByExternalID(ctx, input.ExternalCustomerID)
	return GetOutput{Data: customer}, err
}

func (r orbReader) GetSubscription(ctx context.Context, input GetInput) (GetOutput, error) {
	client, err := r.clientFor(input.Environment)
	if err != nil {
		return GetOutput{}, err
	}
	subscription, err := client.Subscriptions.Fetch(ctx, input.ID)
	return GetOutput{Data: subscription}, err
}

func (r orbReader) GetPlan(ctx context.Context, input GetPlanInput) (GetOutput, error) {
	client, err := r.clientFor(input.Environment)
	if err != nil {
		return GetOutput{}, err
	}
	if input.PlanID != "" {
		plan, err := client.Plans.Fetch(ctx, input.PlanID)
		return GetOutput{Data: plan}, err
	}
	plan, err := client.Plans.ExternalPlanID.Fetch(ctx, input.ExternalPlanID)
	return GetOutput{Data: plan}, err
}

func (r orbReader) GetInvoice(ctx context.Context, input GetInput) (GetOutput, error) {
	client, err := r.clientFor(input.Environment)
	if err != nil {
		return GetOutput{}, err
	}
	invoice, err := client.Invoices.Fetch(ctx, input.ID)
	return GetOutput{Data: invoice}, err
}

func (r orbReader) clientFor(environment Environment) (*orb.Client, error) {
	client, ok := r.clients[environment]
	if !ok {
		return nil, fmt.Errorf("no Orb client configured for %q environment", environment)
	}
	return client, nil
}

func listOutput(data any, hasMore bool, nextCursor string) ListOutput {
	return ListOutput{
		Data: data,
		Pagination: Pagination{
			HasMore:    hasMore,
			NextCursor: nextCursor,
		},
	}
}
