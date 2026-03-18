package tcbs

import (
	"context"
	"fmt"
	"net/url"
)

// GetOrders retrieves all orders for the given account.
func (c *Client) GetOrders(ctx context.Context, accountNo string) (*OrderSearchResponse, error) {
	var resp OrderSearchResponse
	err := c.get(ctx, fmt.Sprintf("/aion/v1/accounts/%s/orders", accountNo), nil, &resp)
	if err != nil {
		return nil, err
	}
	return &resp, nil
}

// GetOrderByID retrieves a specific order by its ID.
func (c *Client) GetOrderByID(ctx context.Context, accountNo, orderID string) (*OrderSearchResponse, error) {
	var resp OrderSearchResponse
	err := c.get(ctx, fmt.Sprintf("/aion/v1/accounts/%s/orders/%s", accountNo, orderID), nil, &resp)
	if err != nil {
		return nil, err
	}
	return &resp, nil
}

// GetMatchingDetails retrieves matching details for the given account.
func (c *Client) GetMatchingDetails(ctx context.Context, accountNo string) (*CommandMatchInformationResponse, error) {
	var resp CommandMatchInformationResponse
	err := c.get(ctx, fmt.Sprintf("/aion/v1/accounts/%s/matching-details", accountNo), nil, &resp)
	if err != nil {
		return nil, err
	}
	return &resp, nil
}

// GetPurchasingPower retrieves purchasing power for the given account.
func (c *Client) GetPurchasingPower(ctx context.Context, accountNo string) (*PurchasingPowerResponse, error) {
	var resp PurchasingPowerResponse
	err := c.get(ctx, fmt.Sprintf("/aion/v1/accounts/%s/ppse", accountNo), nil, &resp)
	if err != nil {
		return nil, err
	}
	return &resp, nil
}

// GetPurchasingPowerBySymbol retrieves purchasing power for a specific symbol.
func (c *Client) GetPurchasingPowerBySymbol(ctx context.Context, accountNo, symbol string) (*PurchasingPowerResponse, error) {
	var resp PurchasingPowerResponse
	err := c.get(ctx, fmt.Sprintf("/aion/v1/accounts/%s/ppse/%s", accountNo, symbol), nil, &resp)
	if err != nil {
		return nil, err
	}
	return &resp, nil
}

// GetPurchasingPowerBySymbolPrice retrieves purchasing power for a specific symbol and price.
func (c *Client) GetPurchasingPowerBySymbolPrice(ctx context.Context, accountNo, symbol, price string) (*PurchasingPowerResponse, error) {
	var resp PurchasingPowerResponse
	err := c.get(ctx, fmt.Sprintf("/aion/v1/accounts/%s/ppse/%s/%s", accountNo, symbol, price), nil, &resp)
	if err != nil {
		return nil, err
	}
	return &resp, nil
}

// GetMarginQuota retrieves margin quota for the given custody ID.
func (c *Client) GetMarginQuota(ctx context.Context, custodyID string) ([]MarginQuotaResponse, error) {
	var resp []MarginQuotaResponse
	err := c.get(ctx, fmt.Sprintf("/aion/v1/customers/%s/accounts", custodyID), nil, &resp)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

// GetMarginAccountInfo retrieves margin account information (debt, RTT, risk ratios).
func (c *Client) GetMarginAccountInfo(ctx context.Context, accountNo string) ([]MarginAccountInfoResponse, error) {
	var resp []MarginAccountInfoResponse
	err := c.get(ctx, fmt.Sprintf("/hydros/v1/account/%s/risk", accountNo), nil, &resp)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

// GetSupplementaryLoanPackages retrieves supplementary loan package details.
func (c *Client) GetSupplementaryLoanPackages(ctx context.Context, accountNo string) (*SupplementaryLoanPackageResponse, error) {
	var resp SupplementaryLoanPackageResponse
	err := c.get(ctx, fmt.Sprintf("/campaign-management/v1/margin/subscription/%s/addons/detail", accountNo), nil, &resp)
	if err != nil {
		return nil, err
	}
	return &resp, nil
}

// GetLoans retrieves the list of loans for the given account.
func (c *Client) GetLoans(ctx context.Context, accountNo string) (*LoanResponse, error) {
	var resp LoanResponse
	err := c.get(ctx, fmt.Sprintf("/khaos/v1/loan/%s", accountNo), nil, &resp)
	if err != nil {
		return nil, err
	}
	return &resp, nil
}

// GetStockAssets retrieves stock asset holdings for the given account.
func (c *Client) GetStockAssets(ctx context.Context, accountNo string) (*SeInfoDTO, error) {
	var resp SeInfoDTO
	err := c.get(ctx, fmt.Sprintf("/aion/v1/accounts/%s/se", accountNo), nil, &resp)
	if err != nil {
		return nil, err
	}
	return &resp, nil
}

// GetCashBalance retrieves cash balance information for the given account.
func (c *Client) GetCashBalance(ctx context.Context, accountNo string) (*CashInvestmentResponse, error) {
	var resp CashInvestmentResponse
	err := c.get(ctx, fmt.Sprintf("/aion/v1/accounts/%s/cashInvestments", accountNo), nil, &resp)
	if err != nil {
		return nil, err
	}
	return &resp, nil
}

// CashStatementParams holds parameters for GetCashStatements.
type CashStatementParams struct {
	AccountNo       string
	FromDate        string // e.g. "2025-01-01"
	ToDate          string // e.g. "2025-01-15"
	PageSize        string
	PageIndex       string
	TransactionCode string
}

// GetCashStatements retrieves cash statement history.
func (c *Client) GetCashStatements(ctx context.Context, params CashStatementParams) (*TransHistCashStatementsResponse, error) {
	query := url.Values{}
	query.Set("accountNo", params.AccountNo)
	query.Set("fromDate", params.FromDate)
	query.Set("toDate", params.ToDate)
	query.Set("pageSize", params.PageSize)
	query.Set("pageIndex", params.PageIndex)
	query.Set("transactionCode", params.TransactionCode)

	var resp TransHistCashStatementsResponse
	err := c.get(ctx, "/erebos/v2/digital/trans-hist-cashStatements", query, &resp)
	if err != nil {
		return nil, err
	}
	return &resp, nil
}

// MarginInfoParams holds parameters for GetMarginInfo.
type MarginInfoParams struct {
	AccountNo  string
	FromDate   string
	ToDate     string
	Page       string
	Size       string
	CustodyCD  string
}

// GetMarginInfo retrieves debt inquiry information.
func (c *Client) GetMarginInfo(ctx context.Context, params MarginInfoParams) (*MarginInfoResponse, error) {
	query := url.Values{}
	query.Set("acctno", params.AccountNo)
	query.Set("fromdate", params.FromDate)
	query.Set("toDate", params.ToDate)
	query.Set("page", params.Page)
	query.Set("size", params.Size)
	query.Set("custodycd", params.CustodyCD)

	var resp MarginInfoResponse
	err := c.get(ctx, "/erebos/v2/digital/margin-info", query, &resp)
	if err != nil {
		return nil, err
	}
	return &resp, nil
}
