package tcbs

import (
	"context"
	"fmt"
	"net/url"
	"strings"
)

// GetStockPrices retrieves stock ticker information and pricing.
// tickers is a comma-separated list of stock symbols.
func (c *Client) GetStockPrices(ctx context.Context, tickers []string) ([]MarketStockInfo, error) {
	query := url.Values{}
	query.Set("tickers", strings.Join(tickers, ","))

	var resp []MarketStockInfo
	err := c.get(ctx, "/tartarus/v1/tickerCommons", query, &resp)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

// GetForeignRoom retrieves foreign investor room information.
// tickers is a comma-separated list of stock symbols.
func (c *Client) GetForeignRoom(ctx context.Context, tickers []string) ([]ForeignRoomInfo, error) {
	query := url.Values{}
	query.Set("tickers", strings.Join(tickers, ","))

	var resp []ForeignRoomInfo
	err := c.get(ctx, "/tartarus/v1/tickerSnaps", query, &resp)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

// GetPutThroughInfo retrieves put-through agreement information.
// tickers is a comma-separated list of stock symbols.
func (c *Client) GetPutThroughInfo(ctx context.Context, tickers []string) ([]PutThroughInfo, error) {
	query := url.Values{}
	query.Set("tickers", strings.Join(tickers, ","))

	var resp []PutThroughInfo
	err := c.get(ctx, "/tartarus/v1/putThroughSnaps", query, &resp)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

// IntradayHistoryParams holds parameters for GetIntradayHistory.
type IntradayHistoryParams struct {
	Ticker string
	Page   int
	Size   int
}

// GetIntradayHistory retrieves intraday price matching history for a ticker.
func (c *Client) GetIntradayHistory(ctx context.Context, params IntradayHistoryParams) (*IntradayHistoryResponse, error) {
	query := url.Values{}
	query.Set("page", fmt.Sprintf("%d", params.Page))
	query.Set("size", fmt.Sprintf("%d", params.Size))

	var resp IntradayHistoryResponse
	err := c.get(ctx, fmt.Sprintf("/nyx/v1/intraday/%s/his/paging", params.Ticker), query, &resp)
	if err != nil {
		return nil, err
	}
	return &resp, nil
}

// GetSupplyDemand retrieves supply and demand data for a ticker.
// investorType is one of: "sheep", "wolf", "shark", "all".
func (c *Client) GetSupplyDemand(ctx context.Context, ticker, investorType string) (*SupplyDemandResponse, error) {
	query := url.Values{}
	if investorType != "" {
		query.Set("type", investorType)
	}

	var resp SupplyDemandResponse
	err := c.get(ctx, fmt.Sprintf("/nyx/v1/intraday/%s/bsa", ticker), query, &resp)
	if err != nil {
		return nil, err
	}
	return &resp, nil
}
