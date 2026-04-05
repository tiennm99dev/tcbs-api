package tcbs

// --- Place Order ---

// PlaceOrderRequest represents a stock order placement request.
type PlaceOrderRequest struct {
	ExecType  string `json:"execType"`
	Price     int    `json:"price"`
	PriceType string `json:"priceType"`
	Quantity  int    `json:"quantity"`
	Symbol    string `json:"symbol"`
}

// PlaceOrderResponse represents the response after placing a stock order.
type PlaceOrderResponse struct {
	Error   string `json:"error"`
	Message string `json:"message"`
	OrderID string `json:"orderId"`
}

// --- Update Order ---

// UpdateOrderRequest represents a stock order update request.
type UpdateOrderRequest struct {
	Price    int `json:"price"`
	Quantity int `json:"quantity"`
}

// UpdateOrderResponse represents the response after updating a stock order.
type UpdateOrderResponse struct {
	Error   string `json:"error"`
	Message string `json:"message"`
	OrderID string `json:"orderId"`
}

// --- Cancel Order ---

// CancelOrderRequest represents a stock order cancellation request.
type CancelOrderRequest struct {
	OrdersList []OrderIDRef `json:"ordersList"`
}

// OrderIDRef represents an order ID reference used in cancel requests.
type OrderIDRef struct {
	OrderID string `json:"orderID"`
}

// CancelOrderResponse represents the response after cancelling stock orders.
type CancelOrderResponse struct {
	Object     string   `json:"object"`
	PageSize   int      `json:"pageSize"`
	PageIndex  int      `json:"pageIndex"`
	TotalCount int      `json:"totalCount"`
	Data       []DataX  `json:"data"`
}

// DataX holds cancel order result details.
type DataX struct {
	Object  string   `json:"object"`
	Details []Detail `json:"details"`
}

// Detail holds a single order cancellation result.
type Detail struct {
	Deleted      string `json:"deleted"`
	ErrorCode    string `json:"errorCode"`
	ErrorMessage string `json:"errorMesage"` // note: typo in spec
	OrderID      string `json:"orderID"`
}

// --- Order Query ---

// OrderSearchResponse represents the order book response.
type OrderSearchResponse struct {
	Object     string      `json:"object"`
	PageSize   int         `json:"pageSize"`
	PageIndex  int         `json:"pageIndex"`
	TotalCount int         `json:"totalCount"`
	Data       []OrderInfo `json:"data"`
}

// OrderInfo represents a single order in the order book.
type OrderInfo struct {
	Object      string  `json:"object"`
	AccountNo   string  `json:"accountNo"`
	OrderID     string  `json:"orderID"`
	ExecType    string  `json:"execType"`
	OrderQtty   float64 `json:"orderQtty"`
	ExecQtty    float64 `json:"execQtty"`
	CodeID      string  `json:"codeID"`
	Symbol      string  `json:"symbol"`
	PriceType   string  `json:"priceType"`
	TxTime      string  `json:"txtime"`
	TxDate      string  `json:"txdate"`
	ExpDate     string  `json:"expDate"`
	TimeType    string  `json:"timeType"`
	OrStatus    string  `json:"orStatus"`
	FeeAcr      float64 `json:"feeAcr"`
	LimitPrice  float64 `json:"limitPrice"`
	CancelQtty  float64 `json:"cancelQtty"`
	RemainQtty  float64 `json:"remainQtty"`
	Via         string  `json:"via"`
	QuotePrice  float64 `json:"quotePrice"`
	MatchPrice  float64 `json:"matchPrice"`
	TradePlace  string  `json:"tradePlace"`
	MatchType   string  `json:"matchType"`
	IsDisposal  string  `json:"isDisposal"`
	IsCancel    string  `json:"isCancel"`
	IsAmend     string  `json:"isAmend"`
	UserName    string  `json:"userName"`
	OrsOrderID  string  `json:"orsOrderID"`
	SecType     string  `json:"sectype"`
	IsFOOrder   string  `json:"isFOOrder"`
	OdTimeStamp string  `json:"odTimeStamp"`
	MatchAmount float64 `json:"matchAmount"`
	MMType      string  `json:"mmType"`
	BRatio      float64 `json:"bRatio"`
	TaxSellAmt  float64 `json:"taxSellAmout"` // note: typo in spec
}

// --- Match Information ---

// CommandMatchInformationResponse represents matching details.
type CommandMatchInformationResponse struct {
	Object     string                           `json:"object"`
	TotalCount int                              `json:"totalCount"`
	PageSize   int                              `json:"pageSize"`
	PageIndex  int                              `json:"pageIndex"`
	Data       []CommandMatchInformationDetail  `json:"data"`
}

// CommandMatchInformationDetail represents a single matching detail.
type CommandMatchInformationDetail struct {
	OrderID    string  `json:"orderId"`
	Side       string  `json:"side"`
	Symbol     string  `json:"symbol"`
	QuoteQtty  float64 `json:"quoteQtty"`
	QuotePrice float64 `json:"quotePrice"`
	TradeID    string  `json:"tradeId"`
	Qtty       float64 `json:"qtty"`
	Price      float64 `json:"price"`
	TimeExec   float64 `json:"timeExec"`
}

// --- Purchasing Power ---

// PurchasingPowerResponse represents purchasing power information.
type PurchasingPowerResponse struct {
	AccountNo        string  `json:"accountNo"`
	CustodyID        string  `json:"custodyID"`
	Symbol           string  `json:"symbol"`
	Price            float64 `json:"price"`
	PP0              float64 `json:"pp0"`
	PPSE             float64 `json:"ppse"`
	PPSERef          float64 `json:"ppseref"`
	MaxBuyQuantity   float64 `json:"maxBuyQuantity"`
	RealMaxBuyQty    float64 `json:"realMaxBuyQuantity"`
	MinBuyQuantity   float64 `json:"minBuyQuantity"`
	MarginRatioLoan  float64 `json:"marginRatioLoan"`
	MarginPriceLoan  float64 `json:"marginPriceLoan"`
	RateBrkS         string  `json:"rateBrkS"`
	RateBrkB         string  `json:"rateBrkB"`
}

// --- Margin ---

// MarginQuotaResponse represents margin quota information.
type MarginQuotaResponse struct {
	CustodyID     string  `json:"custodyID"`
	AccountNo     string  `json:"accountNo"`
	AFType        string  `json:"aftype"`
	VSDStatus     string  `json:"vsdStatus"`
	AccountStatus string  `json:"accountStatus"`
	MarginLimit   float64 `json:"marginLimit"`
	IsIA          string  `json:"isIA"`
	BankName      string  `json:"bankName"`
	BankAccount   string  `json:"bankAccount"`
	AccountType   string  `json:"accountType"`
}

// MarginAccountInfoResponse represents margin account details.
type MarginAccountInfoResponse struct {
	AccountNo       string      `json:"accountNo"`
	RiskPolicy      *RiskPolicy `json:"riskPolicy,omitempty"`
	RTT             float64     `json:"rtt"`
	Outstanding     float64     `json:"outstanding"`
	AccruedInterest float64     `json:"accruedInterest"`
	DueAmount       float64     `json:"dueAmount"`
	OverdueAmount   float64     `json:"overdueAmount"`
	RiskStatus      *RiskStatus `json:"riskStatus,omitempty"`
	TotalFeeDebt    float64     `json:"totalFeeDebt"`
}

// RiskPolicy represents margin risk policy parameters.
type RiskPolicy struct {
	MaintenanceMargin float64 `json:"maintenanceMargin"`
	InitialMargin     float64 `json:"initialMargin"`
	LiquidationMargin float64 `json:"liquidationMargin"`
}

// RiskStatus represents RTT status.
type RiskStatus struct {
	Code        string `json:"code"`
	Description string `json:"description"`
}
