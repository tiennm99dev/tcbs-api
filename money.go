package tcbs

import "context"

// TransferMoney performs an internal money transfer between sub-accounts.
func (c *Client) TransferMoney(ctx context.Context, req *MoneyTransferRequest) (*MoneyTransferResponse, error) {
	var resp MoneyTransferResponse
	err := c.post(ctx, "/physis/v1/stock/transfer", req, &resp)
	if err != nil {
		return nil, err
	}
	return &resp, nil
}

// WithdrawMargin withdraws margin for derivative accounts.
func (c *Client) WithdrawMargin(ctx context.Context, req *MarginDepositWithdrawRequest) (*MarginDepositWithdrawResponse, error) {
	var resp MarginDepositWithdrawResponse
	err := c.post(ctx, "/khronos/v1/cash/withdraw/update", req, &resp)
	if err != nil {
		return nil, err
	}
	return &resp, nil
}

// DepositMargin deposits margin for derivative accounts.
func (c *Client) DepositMargin(ctx context.Context, req *MarginDepositWithdrawRequest) (*MarginDepositWithdrawResponse, error) {
	var resp MarginDepositWithdrawResponse
	err := c.post(ctx, "/khronos/v1/cash/deposit/update", req, &resp)
	if err != nil {
		return nil, err
	}
	return &resp, nil
}
