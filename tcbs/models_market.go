package tcbs

// --- Stock Price (REST) ---

// MarketStockInfo is one row of GET /tartarus/v1/tickerCommons
// (MarketInformationStockNormalResponseDTO). Prices are in VND.
type MarketStockInfo struct {
	Symbol          string `json:"symbol"`
	CeilPrice       Number `json:"ceilPrice"`
	FloorPrice      Number `json:"floorPrice"`
	RefPrice        Number `json:"refPrice"`
	BidPrice01      Number `json:"bidPrice01"`
	BidPrice02      Number `json:"bidPrice02"`
	BidPrice03      Number `json:"bidPrice03"`
	BidQtty01       Number `json:"bidQtty01"`
	BidQtty02       Number `json:"bidQtty02"`
	BidQtty03       Number `json:"bidQtty03"`
	OfferPrice01    Number `json:"offerPrice01"`
	OfferPrice02    Number `json:"offerPrice02"`
	OfferPrice03    Number `json:"offerPrice03"`
	OfferQtty01     Number `json:"offerQtty01"`
	OfferQtty02     Number `json:"offerQtty02"`
	OfferQtty03     Number `json:"offerQtty03"`
	MatchPrice      Number `json:"matchPrice"`
	MatchQtty       Number `json:"matchQtty"`
	Change          Number `json:"change"`
	ChangePercent   Number `json:"changePercent"`
	Open            Number `json:"open"`
	High            Number `json:"high"`
	Low             Number `json:"low"`
	Avg             Number `json:"avg"`
	TotalVol        Number `json:"totalVol"`
	TotalVal        Number `json:"totalVal"`
	OpenVol         Number `json:"openVol"`
	BuyForeignQtty  Number `json:"buyForeignQtty"`
	SellForeignQtty Number `json:"sellForeignQtty"`
	IndexNumber     Number `json:"indexNumber"`
}

// MarketStockInfoResponse is the tickerCommons response envelope.
type MarketStockInfoResponse struct {
	Data        []MarketStockInfo `json:"data"`
	TradingDate string            `json:"tradingDate"` // DD/MM/YYYY
}

// --- Foreign Room (REST) ---

// ForeignRoomInfo represents foreign investor room information.
type ForeignRoomInfo struct {
	Ticker      string  `json:"ticker"`
	TotalRoom   float64 `json:"totalRoom"`
	CurrentRoom float64 `json:"currentRoom"`
	BuyVol      float64 `json:"buyVol"`
	SellVol     float64 `json:"sellVol"`
}

// --- Put-Through (REST) ---

// PutThroughMatchInfo represents put-through match information.
type PutThroughMatchInfo struct {
	Symbol           string  `json:"symbol"`
	Price            float64 `json:"price"`
	Vol              float64 `json:"vol"`
	Val              float64 `json:"val"`
	Time             string  `json:"time"`
	AccumulatedValue float64 `json:"accumulatedValue"`
}

// PutThroughAdvertisementInfo represents put-through advertisement information.
type PutThroughAdvertisementInfo struct {
	Symbol  string  `json:"symbol"`
	Price   float64 `json:"price"`
	Vol     float64 `json:"vol"`
	Time    string  `json:"time"`
	Status  int     `json:"status"`
	Color   int     `json:"color"`
	OrderID string  `json:"orderId"`
	Side    string  `json:"side"`
}

// --- Intraday History ---

// IntradayHistoryResponse represents intraday price matching history.
type IntradayHistoryResponse struct {
	Ticker string                `json:"ticker"`
	Page   int                   `json:"page"`
	Size   int                   `json:"size"`
	Data   []IntradayHistoryItem `json:"data"`
}

// IntradayHistoryItem represents a single intraday trade.
type IntradayHistoryItem struct {
	P   float64 `json:"p"`
	V   float64 `json:"v"`
	CP  float64 `json:"cp"`
	RCP float64 `json:"rcp"`
	A   string  `json:"a"`
	BA  float64 `json:"ba"`
	SA  float64 `json:"sa"`
	HL  bool    `json:"hl"`
	PCP float64 `json:"pcp"`
	T   string  `json:"t"`
}

// --- Supply & Demand ---

// SupplyDemandResponse represents supply and demand data.
type SupplyDemandResponse struct {
	Ticker string             `json:"ticker"`
	Data   []SupplyDemandItem `json:"data"`
}

// SupplyDemandItem represents a single supply/demand data point (bsa-month).
type SupplyDemandItem struct {
	BUP float64 `json:"bup"`
	SDP float64 `json:"sdp"`
	BSR float64 `json:"bsr"`
	T   string  `json:"t"`
}

// SupplyDemand15mItem represents a 15-minute supply/demand data point (bsa, bsa-ext).
type SupplyDemand15mItem struct {
	BU  float64 `json:"bu"`
	BMS float64 `json:"bms"`
	BUP float64 `json:"bup"`
	SD  float64 `json:"sd"`
	SMS string  `json:"sms"`
	SDP float64 `json:"sdp"`
	BSR float64 `json:"bsr"`
	T   string  `json:"t"`
	S   int64   `json:"s"`
}

// SupplyDemand15mResponse wraps a list of 15-minute supply/demand items.
type SupplyDemand15mResponse struct {
	Ticker string                `json:"ticker"`
	Data   []SupplyDemand15mItem `json:"data"`
}

// --- WebSocket Market DTOs ---

// WSStockInfo represents stock information from WebSocket stream.
type WSStockInfo struct {
	Symbol          string  `json:"symbol"`
	CeilPrice       float64 `json:"ceilPrice"`
	FloorPrice      float64 `json:"floorPrice"`
	RefPrice        float64 `json:"refPrice"`
	BidPrice01      float64 `json:"bidPrice01"`
	BidPrice02      float64 `json:"bidPrice02"`
	BidPrice03      float64 `json:"bidPrice03"`
	BidQtty01       float64 `json:"bidQtty01"`
	BidQtty02       float64 `json:"bidQtty02"`
	BidQtty03       float64 `json:"bidQtty03"`
	OfferPrice01    float64 `json:"offerPrice01"`
	OfferPrice02    float64 `json:"offerPrice02"`
	OfferPrice03    float64 `json:"offerPrice03"`
	OfferQtty01     float64 `json:"offerQtty01"`
	OfferQtty02     float64 `json:"offerQtty02"`
	OfferQtty03     float64 `json:"offerQtty03"`
	MatchPrice      float64 `json:"matchPrice"`
	MatchQtty       float64 `json:"matchQtty"`
	Change          float64 `json:"change"`
	ChangePercent   float64 `json:"changePercent"`
	Open            float64 `json:"open"`
	High            float64 `json:"high"`
	Low             float64 `json:"low"`
	TotalVol        float64 `json:"totalVol"`
	TotalVal        float64 `json:"totalVal"`
	OpenVol         float64 `json:"openVol"`
	BuyForeignQtty  float64 `json:"buyForeignQtty"`
	SellForeignQtty float64 `json:"sellForeignQtty"`
	Room            string  `json:"room"`
	Avg             float64 `json:"avg"`
	IndexNumber     float64 `json:"indexNumber"`
}

// WSDerivativeInfo represents derivative information from WebSocket stream.
type WSDerivativeInfo struct {
	Symbol          string  `json:"symbol"`
	CeilPrice       float64 `json:"ceilPrice"`
	FloorPrice      float64 `json:"floorPrice"`
	RefPrice        float64 `json:"refPrice"`
	BidPrice01      float64 `json:"bidPrice01"`
	BidPrice02      float64 `json:"bidPrice02"`
	BidPrice03      float64 `json:"bidPrice03"`
	BidQtty01       float64 `json:"bidQtty01"`
	BidQtty02       float64 `json:"bidQtty02"`
	BidQtty03       float64 `json:"bidQtty03"`
	OfferPrice01    float64 `json:"offerPrice01"`
	OfferPrice02    float64 `json:"offerPrice02"`
	OfferPrice03    float64 `json:"offerPrice03"`
	OfferQtty01     float64 `json:"offerQtty01"`
	OfferQtty02     float64 `json:"offerQtty02"`
	OfferQtty03     float64 `json:"offerQtty03"`
	MatchPrice      float64 `json:"matchPrice"`
	MatchQtty       float64 `json:"matchQtty"`
	Change          float64 `json:"change"`
	ChangePercent   float64 `json:"changePercent"`
	Open            float64 `json:"open"`
	High            float64 `json:"high"`
	Low             float64 `json:"low"`
	TotalVol        float64 `json:"totalVol"`
	OpenVol         float64 `json:"openVol"`
	BuyForeignQtty  float64 `json:"buyForeignQtty"`
	SellForeignQtty float64 `json:"sellForeignQtty"`
	ExpiryDate      string  `json:"expiryDate"`
	Avg             float64 `json:"avg"`
}

// WSForeignIndexInfo represents foreign index information from WebSocket stream.
type WSForeignIndexInfo struct {
	Symbol          string  `json:"symbol"`
	CeilPrice       float64 `json:"ceilPrice"`
	FloorPrice      float64 `json:"floorPrice"`
	RefPrice        float64 `json:"refPrice"`
	BidPrice01      float64 `json:"bidPrice01"`
	BidPrice02      float64 `json:"bidPrice02"`
	BidPrice03      float64 `json:"bidPrice03"`
	BidQtty01       float64 `json:"bidQtty01"`
	BidQtty02       float64 `json:"bidQtty02"`
	BidQtty03       float64 `json:"bidQtty03"`
	OfferPrice01    float64 `json:"offerPrice01"`
	OfferPrice02    float64 `json:"offerPrice02"`
	OfferPrice03    float64 `json:"offerPrice03"`
	OfferQtty01     float64 `json:"offerQtty01"`
	OfferQtty02     float64 `json:"offerQtty02"`
	OfferQtty03     float64 `json:"offerQtty03"`
	MatchPrice      float64 `json:"matchPrice"`
	MatchQtty       float64 `json:"matchQtty"`
	Change          float64 `json:"change"`
	ChangePercent   float64 `json:"changePercent"`
	Open            float64 `json:"open"`
	High            float64 `json:"high"`
	Low             float64 `json:"low"`
	TotalVolume     float64 `json:"totalVolume"`
	TotalValue      float64 `json:"totalValue"`
	BuyForeignQtty  float64 `json:"buyForeignQtty"`
	SellForeignQtty float64 `json:"sellForeignQtty"`
	Room            string  `json:"room"`
	Avg             float64 `json:"avg"`
}
