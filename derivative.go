package tcbs

import (
	"context"
	"fmt"
	"net/url"
	"strings"
)

// GetDerivativeCashStatus retrieves derivative cash and margin overview.
func (c *Client) GetDerivativeCashStatus(ctx context.Context, accountID, subAccountID, getType string) (*DerivativeResponse[*TotalCashDerivativeResponse], error) {
	query := url.Values{}
	query.Set("accountId", accountID)
	query.Set("subAccountId", subAccountID)
	query.Set("getType", getType)

	var resp DerivativeResponse[*TotalCashDerivativeResponse]
	err := c.get(ctx, "/khronos/v1/account/status", query, &resp)
	if err != nil {
		return nil, err
	}
	return &resp, nil
}

// DerivativePositionCloseParams holds parameters for GetDerivativeClosedPositions.
type DerivativePositionCloseParams struct {
	AccountID    string
	SubAccountID string
	Symbol       string // optional
	PageNo       int
	PageSize     int
}

// GetDerivativeClosedPositions retrieves closed derivative positions.
func (c *Client) GetDerivativeClosedPositions(ctx context.Context, params DerivativePositionCloseParams) (*DerivativeResponse[[]AssetPositionCloseDerivativeResponse], error) {
	query := url.Values{}
	query.Set("accountId", params.AccountID)
	query.Set("subAccountId", params.SubAccountID)
	if params.Symbol != "" {
		query.Set("symbol", params.Symbol)
	}
	query.Set("pageNo", fmt.Sprintf("%d", params.PageNo))
	query.Set("pageSize", fmt.Sprintf("%d", params.PageSize))

	var resp DerivativeResponse[[]AssetPositionCloseDerivativeResponse]
	err := c.get(ctx, "/khronos/v1/account/portfolio/position/close", query, &resp)
	if err != nil {
		return nil, err
	}
	return &resp, nil
}

// GetDerivativeOpenPositions retrieves open derivative positions.
func (c *Client) GetDerivativeOpenPositions(ctx context.Context, accountID, subAccountID string) (*DerivativeResponse[[]AssetPositionOpenDerivativeResponse], error) {
	query := url.Values{}
	query.Set("accountId", accountID)
	query.Set("subAccountId", subAccountID)

	var resp DerivativeResponse[[]AssetPositionOpenDerivativeResponse]
	err := c.get(ctx, "/khronos/v1/account/portfolio/status", query, &resp)
	if err != nil {
		return nil, err
	}
	return &resp, nil
}

// DerivativeOrdersParams holds parameters for GetDerivativeNormalOrders.
type DerivativeOrdersParams struct {
	PageNo    int
	PageSize  int
	AccountID string
	Symbol    string // e.g. "ALL,ALL" or "VN30F2303,B"
	RefID     string // optional
	OrderType string // "" for all, "3" arbitrage, "4" SL/TP, "5" force close, "6" normal
	Status    string // "0" all, "1" pending, "2" matched
}

// GetDerivativeNormalOrders retrieves normal derivative orders.
func (c *Client) GetDerivativeNormalOrders(ctx context.Context, params DerivativeOrdersParams) (*DerivativeResponse[[]DerivativeNormalOrderResponse], error) {
	query := url.Values{}
	query.Set("pageNo", fmt.Sprintf("%d", params.PageNo))
	query.Set("pageSize", fmt.Sprintf("%d", params.PageSize))
	query.Set("accountId", params.AccountID)
	query.Set("symbol", params.Symbol)
	if params.RefID != "" {
		query.Set("refId", params.RefID)
	}
	query.Set("orderType", params.OrderType)
	query.Set("status", params.Status)

	var resp DerivativeResponse[[]DerivativeNormalOrderResponse]
	err := c.get(ctx, "/khronos/v1/order/in-day", query, &resp)
	if err != nil {
		return nil, err
	}
	return &resp, nil
}

// GetDerivativeConditionOrders retrieves conditional derivative orders.
func (c *Client) GetDerivativeConditionOrders(ctx context.Context, accountID, subAccountID string, pageNo, pageSize int) (*DerivativeResponse[[]DerivativeConditionOrderResponse], error) {
	query := url.Values{}
	query.Set("accountId", accountID)
	query.Set("subAccountId", subAccountID)
	query.Set("pageNo", fmt.Sprintf("%d", pageNo))
	query.Set("pageSize", fmt.Sprintf("%d", pageSize))

	var resp DerivativeResponse[[]DerivativeConditionOrderResponse]
	err := c.get(ctx, "/khronos/v1/order/condition/detail", query, &resp)
	if err != nil {
		return nil, err
	}
	return &resp, nil
}

// PlaceDerivativeNormalOrder places a normal derivative order.
func (c *Client) PlaceDerivativeNormalOrder(ctx context.Context, req *DerivativeNormalOrderRequest) (*DerivativeResponse[*DerivativeNormalOrderPlaceResponse], error) {
	var resp DerivativeResponse[*DerivativeNormalOrderPlaceResponse]
	err := c.post(ctx, "/khronos/v1/order/place", req, &resp)
	if err != nil {
		return nil, err
	}
	return &resp, nil
}

// PlaceDerivativeConditionOrder places a conditional derivative order (SL/TP, Arbitrage, etc.).
func (c *Client) PlaceDerivativeConditionOrder(ctx context.Context, req *DerivativeConditionOrderRequest) (*DerivativeResponse[*DerivativeConditionOrderPlaceResponse], error) {
	var resp DerivativeResponse[*DerivativeConditionOrderPlaceResponse]
	err := c.post(ctx, "/khronos/v1/order/condition/place", req, &resp)
	if err != nil {
		return nil, err
	}
	return &resp, nil
}

// ChangeDerivativeNormalOrder modifies an existing normal derivative order.
func (c *Client) ChangeDerivativeNormalOrder(ctx context.Context, req *DerivativeChangeNormalOrderRequest) (*DerivativeResponse[string], error) {
	var resp DerivativeResponse[string]
	err := c.post(ctx, "/khronos/v1/order/change", req, &resp)
	if err != nil {
		return nil, err
	}
	return &resp, nil
}

// ChangeDerivativeConditionOrder modifies an existing conditional derivative order.
func (c *Client) ChangeDerivativeConditionOrder(ctx context.Context, req *DerivativeChangeConditionOrderRequest) (*DerivativeResponse[string], error) {
	var resp DerivativeResponse[string]
	err := c.post(ctx, "/khronos/v2/order/condition/change", req, &resp)
	if err != nil {
		return nil, err
	}
	return &resp, nil
}

// CancelDerivativeNormalOrder cancels a normal derivative order.
func (c *Client) CancelDerivativeNormalOrder(ctx context.Context, req *DerivativeCancelNormalOrderRequest) (*DerivativeResponse[*DerivativeCancelNormalOrderResponse], error) {
	var resp DerivativeResponse[*DerivativeCancelNormalOrderResponse]
	err := c.post(ctx, "/khronos/v1/order/cancel", req, &resp)
	if err != nil {
		return nil, err
	}
	return &resp, nil
}

// CancelDerivativeConditionOrder cancels a conditional derivative order.
func (c *Client) CancelDerivativeConditionOrder(ctx context.Context, req *DerivativeCancelConditionOrderRequest) (*DerivativeResponse[string], error) {
	var resp DerivativeResponse[string]
	err := c.post(ctx, "/khronos/v1/order/condition/cancel", req, &resp)
	if err != nil {
		return nil, err
	}
	return &resp, nil
}

// GetDerivativeMarketInfo retrieves derivative contract pricing and information.
func (c *Client) GetDerivativeMarketInfo(ctx context.Context, tickers []string) ([]DerivativeMarketInfo, error) {
	query := url.Values{}
	query.Set("tickers", strings.Join(tickers, ","))

	var resp []DerivativeMarketInfo
	err := c.get(ctx, "/tartarus/v1/derivatives", query, &resp)
	if err != nil {
		return nil, err
	}
	return resp, nil
}
