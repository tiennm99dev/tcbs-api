package tcbs

import (
	"context"
	"fmt"
)

// PlaceOrder places a stock order on the given account.
func (c *Client) PlaceOrder(ctx context.Context, accountNo string, req *PlaceOrderRequest) (*PlaceOrderResponse, error) {
	var resp PlaceOrderResponse
	err := c.post(ctx, fmt.Sprintf("/akhlys/v1/accounts/%s/orders", accountNo), req, &resp)
	if err != nil {
		return nil, err
	}
	return &resp, nil
}

// UpdateOrder modifies an existing stock order.
func (c *Client) UpdateOrder(ctx context.Context, accountNo, orderID string, req *UpdateOrderRequest) (*UpdateOrderResponse, error) {
	var resp UpdateOrderResponse
	err := c.put(ctx, fmt.Sprintf("/akhlys/v1/accounts/%s/orders/%s", accountNo, orderID), req, &resp)
	if err != nil {
		return nil, err
	}
	return &resp, nil
}

// CancelOrder cancels an existing stock order.
func (c *Client) CancelOrder(ctx context.Context, accountNo string, req *CancelOrderRequest) (*CancelOrderResponse, error) {
	var resp CancelOrderResponse
	err := c.put(ctx, fmt.Sprintf("/akhlys/v1/accounts/%s/cancel-orders", accountNo), req, &resp)
	if err != nil {
		return nil, err
	}
	return &resp, nil
}
