package tcbs

// DerivativeResponse is a generic wrapper for derivative API responses.
type DerivativeResponse[T any] struct {
	Cmd  string `json:"cmd"`
	RC   string `json:"rc"`
	RS   string `json:"rs"`
	OID  string `json:"oID"`
	Data T      `json:"data"`
}

// --- Cash & Positions ---

// TotalCashDerivativeResponse represents derivative cash/margin overview.
type TotalCashDerivativeResponse struct {
	Cash              float64 `json:"cash"`
	Stock             float64 `json:"stock"`
	Collateral        float64 `json:"collateral"`
	Type              string  `json:"type"`
	Net               string  `json:"net"`
	Tyle              string  `json:"tyle"`
	IM                float64 `json:"im"`
	VM                float64 `json:"vm"`
	DM                float64 `json:"dm"`
	MR                float64 `json:"mr"`
	AvaiCash          float64 `json:"avaiCash"`
	AvaiColla         float64 `json:"avaiColla"`
	VMUnpay           float64 `json:"vmunpay"`
	Info              string  `json:"info"`
	Color             string  `json:"color"`
	VMEod             string  `json:"vm_eod"`
	Others            float64 `json:"others"`
	Tax               float64 `json:"tax"`
	FeeCTCK           float64 `json:"feeCTCK"`
	FeeHNX            float64 `json:"feeHNX"`
	CashWithdraw      float64 `json:"cashWithdraw"`
	TienBoSung        float64 `json:"tienbosung"`
	CashAvailWithdraw float64 `json:"cashavaiwithdraw"`
	Assets            float64 `json:"assets"`
	NAV               float64 `json:"nav"`
	CashOut           float64 `json:"cashOut"`
	UnrealizeVM       float64 `json:"unrelizeVM"`
	FeePos            float64 `json:"feePos"`
	FeeMan            float64 `json:"feeMan"`
	Product           string  `json:"product"`
	Status            string  `json:"status"`
	Debt              string  `json:"debt"`
	W1                float64 `json:"w1"`
	W2                float64 `json:"w2"`
	Limit             float64 `json:"limit"`
	Package           string  `json:"package"`
}

// AssetPositionCloseDerivativeResponse represents a closed derivative position.
type AssetPositionCloseDerivativeResponse struct {
	Symbol        string `json:"symbol"`
	Side          string `json:"side"`
	OpenPrice     string `json:"openPrice"`
	ClosePrice    string `json:"closePrice"`
	ClosePosition string `json:"closePosition"`
	Fee           string `json:"fee"`
	Tax           string `json:"tax"`
	CloseVM       string `json:"closeVM"`
	Unrealize     string `json:"unrealize"`
	ClosePC       string `json:"closePC"`
	Time          string `json:"time"`
}

// AssetPositionOpenDerivativeResponse represents an open derivative position.
type AssetPositionOpenDerivativeResponse struct {
	Symbol     string  `json:"symbol"`
	IM         string  `json:"im"`
	Deliver    int     `json:"deliver"`
	Receive    int     `json:"receive"`
	Net        int     `json:"net"`
	Side       string  `json:"side"`
	Account    string  `json:"account"`
	WASP       float64 `json:"wasp"`
	WAPB       float64 `json:"wapb"`
	LastPrice  float64 `json:"lastPrice"`
	IMValue    float64 `json:"imValue"`
	VMValue    float64 `json:"vmValue"`
	MRValue    float64 `json:"mrValue"`
	DueDate    string  `json:"duedate"`
	NetOffVol  int     `json:"netoffvol"`
	AvgRemain  float64 `json:"avg_remain"`
	VMRemain   float64 `json:"vm_remain"`
	PCRemain   float64 `json:"pc_remain"`
	StopLoss   string  `json:"stoploss"`
	TakeProfit string  `json:"takeprofit"`
}

// --- Normal Orders ---

// DerivativeNormalOrderRequest represents a request to place a normal derivative order.
type DerivativeNormalOrderRequest struct {
	AccountID    string  `json:"accountId"`
	SubAccountID string  `json:"subAccountId"`
	Side         string  `json:"side"`
	Symbol       string  `json:"symbol"`
	Price        float64 `json:"price"`
	Volume       int     `json:"volume"`
	Advance      string  `json:"advance,omitempty"`
	RefID        string  `json:"refId,omitempty"`
	OrderType    string  `json:"orderType"`
	Pin          string  `json:"pin,omitempty"`
}

// DerivativeNormalOrderResponse represents a normal derivative order in list responses.
type DerivativeNormalOrderResponse struct {
	OrderNo     string  `json:"orderNo"`
	PKOrderNo   string  `json:"pk_orderNo"`
	RefID       string  `json:"refId"`
	OrderTime   string  `json:"orderTime"`
	AccountCode string  `json:"accountCode"`
	Side        string  `json:"side"`
	Symbol      string  `json:"symbol"`
	Volume      float64 `json:"volume"`
	ShowPrice   float64 `json:"showPrice"`
	MatchVolume float64 `json:"matchVolume"`
	MatchPriceBQ float64 `json:"matchPriceBQ"`
	Status      string  `json:"status"`
	OrderStatus string  `json:"orderStatus"`
	Channel     string  `json:"channel"`
	Group       string  `json:"group"`
	CancelTime  string  `json:"cancelTime"`
	IsCancel    float64 `json:"isCancel"`
	IsAmend     float64 `json:"isAmend"`
	Info        string  `json:"info"`
	MaxPrice    float64 `json:"maxPrice"`
	MatchValue  float64 `json:"matchValue"`
	Quote       string  `json:"quote"`
	AutoType    string  `json:"autoType"`
	Product     string  `json:"product"`
	OrderType   string  `json:"orderType"`
	Source      string  `json:"source"`
}

// DerivativeNormalOrderPlaceResponse represents the response after placing a normal order.
type DerivativeNormalOrderPlaceResponse struct {
	Symbol      string  `json:"symbol"`
	ShareStatus string  `json:"shareStatus"`
	Status      string  `json:"status"`
	MsgType     string  `json:"msg_type"`
	ShowPrice   float64 `json:"showPrice"`
	OrderTime   string  `json:"orderTime"`
	Type        string  `json:"type"`
	AccountCode string  `json:"accountCode"`
	OrderNo     string  `json:"orderNo"`
	Market      string  `json:"market"`
	MatchVolume float64 `json:"matchVolume"`
	Side        string  `json:"side"`
	Volume      float64 `json:"volume"`
	PKOrderNo   string  `json:"pk_orderNo"`
	Channel     string  `json:"channel"`
	RefID       string  `json:"refID"`
	Group       string  `json:"group"`
	AccType     string  `json:"accType"`
	Quote       string  `json:"quote"`
	AutoType    string  `json:"autoType"`
	Product     string  `json:"product"`
}

// --- Condition Orders ---

// DerivativeConditionOrderRequest represents a request to place a conditional order.
type DerivativeConditionOrderRequest struct {
	AccountID       string  `json:"accountId"`
	SubAccountID    string  `json:"subAccountId"`
	Side            string  `json:"side"`
	Symbol          string  `json:"symbol"`
	Price           float64 `json:"price"`
	Volume          float64 `json:"volume"`
	Advance         string  `json:"advance,omitempty"`
	RefID           string  `json:"refId,omitempty"`
	OrderType       string  `json:"orderType"`
	Pin             string  `json:"pin,omitempty"`
	Type            string  `json:"type,omitempty"`
	Cmd             string  `json:"cmd,omitempty"`
	CallbackPoint   float64 `json:"callbackPoint,omitempty"`
	ActivationPrice float64 `json:"activationPrice,omitempty"`
	SOPrice         float64 `json:"soPrice,omitempty"`
}

// DerivativeConditionOrderResponse represents a conditional order in list responses.
type DerivativeConditionOrderResponse struct {
	OrderNo    string  `json:"orderNo"`
	GroupOrder string  `json:"groupOrder"`
	PKOrderNo  string  `json:"pk_orderNo"`
	AccountCode string `json:"accountCode"`
	Side       string  `json:"side"`
	Symbol     string  `json:"symbol"`
	ShowPrice  float64 `json:"showPrice"`
	Volume     float64 `json:"volume"`
	Condition  string  `json:"condition"`
	Result     string  `json:"result"`
	ActiveTime string  `json:"active_time"`
	SendTime   string  `json:"send_time"`
	CancelTime string  `json:"cancel_time"`
	Group      string  `json:"group"`
	Channel    string  `json:"channel"`
	MaxPrice   string  `json:"maxPrice"`
	SOPrice    float64 `json:"soPrice"`
	OrderType  string  `json:"orderType"`
	FromTime   string  `json:"from_time"`
	ExpTime    string  `json:"exp_time"`
	Status     string  `json:"status"`
	Details    string  `json:"details"`
	Notes      string  `json:"notes"`
}

// DerivativeConditionOrderPlaceResponse represents the response after placing a condition order.
type DerivativeConditionOrderPlaceResponse struct {
	Symbol      string  `json:"symbol"`
	ShareStatus string  `json:"shareStatus"`
	Status      string  `json:"status"`
	MsgType     string  `json:"msg_type"`
	ShowPrice   float64 `json:"showPrice"`
	OrderTime   string  `json:"orderTime"`
	Type        string  `json:"type"`
	AccountCode string  `json:"accountCode"`
	OrderNo     int     `json:"orderNo"`
	Market      string  `json:"market"`
	MatchVolume float64 `json:"matchVolume"`
	Side        string  `json:"side"`
	Volume      float64 `json:"volume"`
	PKOrderNo   string  `json:"pk_orderNo"`
	Channel     string  `json:"channel"`
	RefID       string  `json:"refID"`
	Group       string  `json:"group"`
	AccType     string  `json:"accType"`
	Quote       string  `json:"quote"`
	AutoType    string  `json:"autoType"`
	Product     string  `json:"product"`
}

// --- Edit Orders ---

// DerivativeChangeNormalOrderRequest represents a request to modify a normal order.
type DerivativeChangeNormalOrderRequest struct {
	AccountID    string  `json:"accountId"`
	SubAccountID string  `json:"subAccountId"`
	OrderNo      string  `json:"orderNo"`
	RefID        string  `json:"refId"`
	NVol         float64 `json:"nvol"`
	NPrice       float64 `json:"nprice"`
}

// DerivativeChangeConditionOrderRequest represents a request to modify a conditional order.
type DerivativeChangeConditionOrderRequest struct {
	AccountID string  `json:"accountId"`
	PKOrderNo string  `json:"pkOrderNo"`
	Type      string  `json:"type"`
	RefID     string  `json:"refId"`
	SOPrice   float64 `json:"soPrice"`
	Cmd       string  `json:"cmd"`
}

// --- Cancel Orders ---

// DerivativeCancelNormalOrderRequest represents a request to cancel a normal order.
type DerivativeCancelNormalOrderRequest struct {
	AccountID string `json:"accountId"`
	OrderNo   string `json:"orderNo"`
	Cmd       string `json:"cmd"`
	Pin       string `json:"pin,omitempty"`
	RefID     string `json:"refId,omitempty"`
}

// DerivativeCancelNormalOrderResponse represents the response after cancelling a normal order.
type DerivativeCancelNormalOrderResponse struct {
	OrderNo    string `json:"orderNo"`
	MsgType    string `json:"msg_type"`
	Status     string `json:"status"`
	PKOrderNo  string `json:"pk_orderNo"`
	CancelTime string `json:"cancelTime"`
}

// DerivativeCancelConditionOrderRequest represents a request to cancel a conditional order.
type DerivativeCancelConditionOrderRequest struct {
	AccountID    string `json:"accountId"`
	SubAccountID string `json:"subAccountId"`
	OrderNo      string `json:"orderNo"`
}

// --- Market Info ---

// DerivativeMarketInfo represents derivative contract pricing from REST API.
type DerivativeMarketInfo struct {
	Ticker       string  `json:"ticker"`
	RefPrice     float64 `json:"refPrice"`
	CeilingPrice float64 `json:"ceilingPrice"`
	FloorPrice   float64 `json:"floorPrice"`
	HighPrice    float64 `json:"highPrice"`
	LowPrice     float64 `json:"lowPrice"`
	LastPrice    float64 `json:"lastPrice"`
	LastVol      float64 `json:"lastVol"`
	TotalVol     float64 `json:"totalVol"`
	OpenInterest float64 `json:"openInterest"`
}

// OrderIDResponse represents a generic order ID response from derivative endpoints.
type OrderIDResponse struct {
	OrderID string `json:"orderID"`
}
