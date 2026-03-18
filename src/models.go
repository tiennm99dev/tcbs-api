package tcbs

// DerivativeResponse is a generic wrapper for derivative API responses.
type DerivativeResponse[T any] struct {
	Cmd  string `json:"cmd"`
	RC   string `json:"rc"`
	RS   string `json:"rs"`
	OID  string `json:"oID"`
	Data T      `json:"data"`
}

// --- Account Models ---

// AccountInformationResponse represents sub-account information.
type AccountInformationResponse struct {
	BasicInfo       *BasicInfo       `json:"basicInfo,omitempty"`
	PersonalInfo    *PersonalInfo    `json:"personalInfo,omitempty"`
	BankSubAccounts []BankSubAccount `json:"bankSubAccounts,omitempty"`
	BankAccounts    []BankAccount    `json:"bankAccounts,omitempty"`
}

// BasicInfo holds basic account information.
type BasicInfo struct {
	TcbsID     string `json:"tcbsId"`
	Code105C   string `json:"code105C"`
	Status     string `json:"status"`
	FullName   string `json:"fullName"`
	Email      string `json:"email"`
	Phone      string `json:"phone"`
	CustodyCD  string `json:"custodycd"`
	BranchCode string `json:"branchCode"`
}

// PersonalInfo holds personal information.
type PersonalInfo struct {
	IDNumber    string `json:"idNumber"`
	IDIssueDate string `json:"idIssueDate"`
	IDPlace     string `json:"idPlace"`
	DateOfBirth string `json:"dateOfBirth"`
	Gender      string `json:"gender"`
	Address     string `json:"address"`
}

// BankSubAccount represents a sub-account linked to a bank.
type BankSubAccount struct {
	AccountNo   string `json:"accountNo"`
	AccountType string `json:"accountType"`
	Status      string `json:"status"`
}

// BankAccount represents a linked bank account.
type BankAccount struct {
	BankName    string `json:"bankName"`
	BankAccount string `json:"bankAccount"`
	BankBranch  string `json:"bankBranch"`
	IsDefault   string `json:"isDefault"`
}

// --- Order Models ---

// PlaceOrderRequest represents a stock order placement request.
type PlaceOrderRequest struct {
	Symbol    string  `json:"symbol"`
	ExecType  string  `json:"execType"`
	OrderQtty float64 `json:"orderQtty"`
	Price     float64 `json:"price"`
	PriceType string  `json:"priceType"`
	Via       string  `json:"via,omitempty"`
}

// PlaceOrderResponse represents the response after placing a stock order.
type PlaceOrderResponse struct {
	Object  string `json:"object"`
	OrderID string `json:"orderID"`
	Status  string `json:"status"`
}

// UpdateOrderRequest represents a stock order update request.
type UpdateOrderRequest struct {
	OrderQtty float64 `json:"orderQtty"`
	Price     float64 `json:"price"`
	PriceType string  `json:"priceType"`
}

// UpdateOrderResponse represents the response after updating a stock order.
type UpdateOrderResponse struct {
	Object  string `json:"object"`
	OrderID string `json:"orderID"`
	Status  string `json:"status"`
}

// CancelOrderRequest represents a stock order cancellation request.
type CancelOrderRequest struct {
	OrderID string `json:"orderID"`
}

// CancelOrderResponse represents the response after cancelling a stock order.
type CancelOrderResponse struct {
	Object  string `json:"object"`
	OrderID string `json:"orderID"`
	Status  string `json:"status"`
}

// OrderSearchResponse represents the order book response.
type OrderSearchResponse struct {
	Object     string      `json:"object"`
	PageSize   int         `json:"pageSize"`
	PageIndex  string      `json:"pageIndex"`
	TotalCount int64       `json:"totalCount"`
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
	BRatio      float64 `json:"bRatio"`
	TaxSellAmt  float64 `json:"taxSellAmout"`
}

// CommandMatchInformationResponse represents matching details.
type CommandMatchInformationResponse struct {
	Object     string                    `json:"object"`
	PageSize   int                       `json:"pageSize"`
	PageIndex  string                    `json:"pageIndex"`
	TotalCount int64                     `json:"totalCount"`
	Data       []CommandMatchInformation `json:"data"`
}

// CommandMatchInformation represents a single matching detail.
type CommandMatchInformation struct {
	Object     string  `json:"object"`
	OrderID    string  `json:"orderID"`
	AccountNo  string  `json:"accountNo"`
	Symbol     string  `json:"symbol"`
	ExecType   string  `json:"execType"`
	MatchQtty  float64 `json:"matchQtty"`
	MatchPrice float64 `json:"matchPrice"`
	MatchDate  string  `json:"matchDate"`
	MatchTime  string  `json:"matchTime"`
	PriceType  string  `json:"priceType"`
}

// --- Purchasing Power Models ---

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

// --- Asset Models ---

// SeInfoDTO represents stock asset information.
type SeInfoDTO struct {
	Object    string             `json:"object"`
	AccountNo string            `json:"accountNo"`
	CustodyID string            `json:"custodyID"`
	FullName  string            `json:"fullName"`
	Stock     []StockHoldingInfo `json:"stock"`
}

// StockHoldingInfo represents a single stock holding.
type StockHoldingInfo struct {
	Symbol            string  `json:"symbol"`
	SecType           string  `json:"secType"`
	SecTypeName       string  `json:"secTypeName"`
	AvailableTrading  float64 `json:"availableTrading"`
	Mortgaged         float64 `json:"mortgaged"`
	T0                float64 `json:"t0"`
	T1                float64 `json:"t1"`
	T2                float64 `json:"t2"`
	Blocked           float64 `json:"blocked"`
	SecuredQuantity   float64 `json:"securedQuantity"`
	SellRemain        float64 `json:"sellRemain"`
	ExercisedCA       float64 `json:"exercisedCA"`
	UnexercisedCA     float64 `json:"unexercisedCA"`
	StockDividend     float64 `json:"stockDividend"`
	CashDividend      float64 `json:"cashDividend"`
	WaitForTrade      float64 `json:"waitForTrade"`
	WaitForTransfer   float64 `json:"waitForTransfer"`
	WaitForWithdraw   float64 `json:"waitForWithdraw"`
	CurrentPrice      float64 `json:"currentPrice"`
	CostPrice         float64 `json:"costPrice"`
	SellExec          float64 `json:"sellExec"`
	OnHold            float64 `json:"onHold"`
	TotalQtty         float64 `json:"totalQtty"`
	Settlement        float64 `json:"settlement"`
}

// CashInvestmentResponse represents cash balance information.
type CashInvestmentResponse struct {
	Object     string             `json:"object"`
	TotalCount int                `json:"totalCount"`
	PageSize   int                `json:"pageSize"`
	PageIndex  int                `json:"pageIndex"`
	Data       []CashInvestment   `json:"data"`
}

// CashInvestment represents a single cash investment record.
type CashInvestment struct {
	Object           string   `json:"object"`
	IAInfos          []IAInfo `json:"iaInfos"`
	PP0ForBF         float64  `json:"pp0forBF"`
	BankAvlBalanceBF float64  `json:"bankAvlBalanceBF"`
	BodBalance       float64  `json:"bodBalance"`
	CashBalance      float64  `json:"cashBalance"`
	AccountNo        string   `json:"accountNo"`
	CustodyID        string   `json:"custodyID"`
	FullName         string   `json:"fullName"`
	Balance          float64  `json:"balance"`
	AvlAdvanceAmount float64  `json:"avlAdvanceAmount"`
	BuyingAmount     float64  `json:"buyingAmount"`
	BlockAmount      float64  `json:"blockAmount"`
	CashDividend     float64  `json:"cashDevident"`
	BankAvlBalance   float64  `json:"bankAvlBalance"`
	BankBlockAmount  float64  `json:"bankBlockAmount"`
	AvlWithdraw      float64  `json:"avlWithdraw"`
	PP0              float64  `json:"pp0"`
	SecureAmtPO      float64  `json:"secureAmtPO"`
	BondBlockAmount  float64  `json:"bondBlockAmount"`
	MBlockAmount     float64  `json:"mBlockAmount"`
	FundBlockAmount  float64  `json:"fundBlockAmount"`
	AvalBondBlock    float64  `json:"avalBondBlockAmount"`
	DepoFee          float64  `json:"depoFee"`
	BCashDividend    float64  `json:"bCashDividend"`
	SCashDividend    float64  `json:"sCashDividend"`
	DSecured         float64  `json:"dsecured"`
	AdUsed           float64  `json:"adused"`
	MrUsed           float64  `json:"mrused"`
}

// IAInfo represents instant account (IA) source information.
type IAInfo struct {
	Partner   string  `json:"partner"`
	Available float64 `json:"available"`
	Hold      float64 `json:"hold"`
}

// TransHistCashStatementsResponse represents cash statement history.
type TransHistCashStatementsResponse struct {
	Response *TransHistCashStatementsData `json:"response"`
}

// TransHistCashStatementsData holds the paged data of cash statements.
type TransHistCashStatementsData struct {
	PageIndex        int                  `json:"pageIndex"`
	PageSize         int                  `json:"pageSize"`
	TotalCreditAmt   int64                `json:"totalCreditAmount"`
	TotalDebitAmt    int64                `json:"totalDebitAmount"`
	TotalCount       int                  `json:"totalCount"`
	Data             []CashStatementEntry `json:"data"`
}

// CashStatementEntry represents a single cash statement entry.
type CashStatementEntry struct {
	CustodyID       string  `json:"custodyID"`
	TransactionCode string  `json:"transactionCode"`
	DebitAmount     float64 `json:"debitAmount"`
	TransactionName string  `json:"transactionName"`
	Descriptions    string  `json:"descriptions"`
	BusinessDate    string  `json:"businessDate"`
	TransactionNum  string  `json:"transactionNum"`
	AccountNo       string  `json:"accountNo"`
	TransactionDate string  `json:"transationDate"`
	CreditAmount    float64 `json:"creditAmount"`
}

// MarginInfoResponse represents debt inquiry response.
type MarginInfoResponse struct {
	Response *MarginInfoData `json:"response"`
}

// MarginInfoData holds paged margin info data.
type MarginInfoData struct {
	TotalRow  int              `json:"totalRow"`
	TotalPage int              `json:"totalPage"`
	Data      []MarginInfoItem `json:"data"`
}

// MarginInfoItem represents a single margin/debt record.
type MarginInfoItem struct {
	RemainingInterestFee float64 `json:"remainingInterestFee"`
	ReleasedDay          int     `json:"releasedDay"`
	PrintAmount          float64 `json:"printAmount"`
	PaidInterestFee      float64 `json:"paidInterestFee"`
	IntAmount            float64 `json:"intAmount"`
	ReleaseDate          string  `json:"releaseDate"`
	Rate2                float64 `json:"rate2"`
	OverDueDate          string  `json:"overDueDate"`
	PaidFee              float64 `json:"paidFee"`
	ReleasedAmount       float64 `json:"releasedAmount"`
	RemainingFee         float64 `json:"remainingFee"`
	IntPaid              float64 `json:"intPaid"`
	PrinPaid             float64 `json:"prinPaid"`
}

// SupplementaryLoanPackageResponse represents supplementary loan package info.
type SupplementaryLoanPackageResponse struct {
	MarginSureViews []MarginSureView `json:"marginSureViews"`
	TPlus           *TPlusData       `json:"tplus,omitempty"`
}

// MarginSureView represents a margin-sure insurance package.
type MarginSureView struct {
	ID              float64             `json:"id"`
	Name            string              `json:"name"`
	Code            string              `json:"code"`
	SubscriptionFee float64             `json:"subscriptionFee"`
	Status          string              `json:"status"`
	Proposals       []MarginSureProposal `json:"proposals"`
	Default         bool                `json:"default"`
}

// MarginSureProposal represents a proposal within a margin-sure package.
type MarginSureProposal struct {
	ID                       float64 `json:"id"`
	MarginInsuranceID        float64 `json:"marginInsuranceId"`
	InterestAdjustmentValue  float64 `json:"interestAdjustmentValue"`
	InterestPercentThreshold float64 `json:"interestPercentThreshold"`
	ThresholdType            string  `json:"thresholdType"`
}

// TPlusData contains T+ loan package info.
type TPlusData struct {
	Data []TPlusPackage `json:"data"`
}

// TPlusPackage represents a single T+ loan package.
type TPlusPackage struct {
	FirstRate                                float64           `json:"firstRate"`
	ID                                       float64           `json:"id"`
	Name                                     string            `json:"name"`
	Status                                   string            `json:"status"`
	UndueInterestType                        string            `json:"undueInterestType"`
	UndueLadderValue                         []TPlusLadder     `json:"undueLadderValue"`
	OverdueInterest                          float64           `json:"overdueInterest"`
	ExtensionInterest                        float64           `json:"extensionInterest"`
	ExtensionInterestBeforeInterestSettlement float64          `json:"extensionInterestBeforeInterestSettlement"`
	InterestCalculationBasis                 float64           `json:"interestCalculationBasis"`
	UndueFee                                 float64           `json:"undueFee"`
	OverdueFee                               float64           `json:"overdueFee"`
	ExtensionFee                             float64           `json:"extensionFee"`
	DebtCollectionFee                        float64           `json:"debtCollectionFee"`
	Description                              string            `json:"description"`
	ValidFrom                                string            `json:"validFrom"`
}

// TPlusLadder represents a ladder interest rate tier.
type TPlusLadder struct {
	ID        float64 `json:"id"`
	Rate      float64 `json:"rate"`
	StartDate float64 `json:"startDate"`
	DueDate   float64 `json:"dueDate"`
}

// LoanResponse represents the loan list response.
type LoanResponse struct {
	Size    int        `json:"size"`
	Content []LoanItem `json:"content"`
}

// LoanItem represents a single loan.
type LoanItem struct {
	OpeningDate        string     `json:"openingDate"`
	DueDate            string     `json:"dueDate"`
	RenewTime          int        `json:"renewTime"`
	MaxRenewTime       int        `json:"maxRenewTime"`
	IsRenewable        bool       `json:"isRenewable"`
	ReasonList         []string   `json:"reasonList"`
	Symbol             string     `json:"symbol"`
	ID                 float64    `json:"id"`
	AccountNo          string     `json:"accountNo"`
	Principal          float64    `json:"principal"`
	RemainingPrincipal float64    `json:"remainingPrincipal"`
	Interest           float64    `json:"interest"`
	Rate               float64    `json:"rate"`
	Status             string     `json:"status"`
	LoanDays           int        `json:"loanDays"`
	MrxLoanID          float64    `json:"mrxLoanId"`
	Fee                float64    `json:"fee"`
	UndueLoanFee       float64    `json:"undueLoanFee"`
	PricingPolicyType  string     `json:"pricingPolicyType"`
}

// --- Money Management Models ---

// MoneyTransferRequest represents an internal money transfer request.
type MoneyTransferRequest struct {
	SenderAccount   string  `json:"senderAccount"`
	ReceiverAccount string  `json:"receiverAccount"`
	Amount          float64 `json:"amount"`
}

// MoneyTransferResponse represents the transfer response.
type MoneyTransferResponse struct {
	Status  string `json:"status"`
	Message string `json:"message"`
}

// MarginDepositWithdrawRequest represents a margin deposit or withdrawal request.
type MarginDepositWithdrawRequest struct {
	AccountID    string  `json:"accountId"`
	SubAccountID string  `json:"subAccountId"`
	Amount       float64 `json:"amount"`
}

// MarginDepositWithdrawResponse represents the deposit/withdraw response.
type MarginDepositWithdrawResponse struct {
	Cmd  string `json:"cmd"`
	RC   string `json:"rc"`
	RS   string `json:"rs"`
	OID  string `json:"oID"`
}

// --- Market Information Models ---

// MarketStockInfo represents stock ticker information.
type MarketStockInfo struct {
	Ticker          string  `json:"ticker"`
	Exchange        string  `json:"exchange"`
	RefPrice        float64 `json:"refPrice"`
	CeilingPrice    float64 `json:"ceilingPrice"`
	FloorPrice      float64 `json:"floorPrice"`
	HighPrice       float64 `json:"highPrice"`
	LowPrice        float64 `json:"lowPrice"`
	MatchPrice      float64 `json:"matchPrice"`
	MatchQtty       float64 `json:"matchQtty"`
	TotalMatchQtty  float64 `json:"totalMatchQtty"`
	TotalMatchValue float64 `json:"totalMatchValue"`
	Best1BidPrice   float64 `json:"best1BidPrice"`
	Best1BidQtty    float64 `json:"best1BidQtty"`
	Best2BidPrice   float64 `json:"best2BidPrice"`
	Best2BidQtty    float64 `json:"best2BidQtty"`
	Best3BidPrice   float64 `json:"best3BidPrice"`
	Best3BidQtty    float64 `json:"best3BidQtty"`
	Best1OfferPrice float64 `json:"best1OfferPrice"`
	Best1OfferQtty  float64 `json:"best1OfferQtty"`
	Best2OfferPrice float64 `json:"best2OfferPrice"`
	Best2OfferQtty  float64 `json:"best2OfferQtty"`
	Best3OfferPrice float64 `json:"best3OfferPrice"`
	Best3OfferQtty  float64 `json:"best3OfferQtty"`
}

// ForeignRoomInfo represents foreign investor room information.
type ForeignRoomInfo struct {
	Ticker        string  `json:"ticker"`
	TotalRoom     float64 `json:"totalRoom"`
	CurrentRoom   float64 `json:"currentRoom"`
	BuyVol        float64 `json:"buyVol"`
	SellVol       float64 `json:"sellVol"`
}

// PutThroughInfo represents put-through agreement information.
type PutThroughInfo struct {
	Ticker  string  `json:"ticker"`
	Vol     float64 `json:"vol"`
	Val     float64 `json:"val"`
}

// IntradayHistoryResponse represents intraday price matching history.
type IntradayHistoryResponse struct {
	Ticker string              `json:"ticker"`
	Page   int                 `json:"page"`
	Size   int                 `json:"size"`
	Data   []IntradayHistoryItem `json:"data"`
}

// IntradayHistoryItem represents a single intraday trade.
type IntradayHistoryItem struct {
	P  float64 `json:"p"`
	V  float64 `json:"v"`
	CP float64 `json:"cp"`
	RCP float64 `json:"rcp"`
	A  string  `json:"a"`
	BA string  `json:"ba"`
	SA string  `json:"sa"`
	HL string  `json:"hl"`
	PCP float64 `json:"pcp"`
	T  string  `json:"t"`
}

// SupplyDemandResponse represents supply and demand data.
type SupplyDemandResponse struct {
	Ticker string              `json:"ticker"`
	Data   []SupplyDemandItem `json:"data"`
}

// SupplyDemandItem represents a single supply/demand data point.
type SupplyDemandItem struct {
	BU  float64 `json:"bu"`
	BMS float64 `json:"bms"`
	BUP float64 `json:"bup"`
	SD  float64 `json:"sd"`
	SMS float64 `json:"sms"`
	SDP float64 `json:"sdp"`
	BSR float64 `json:"bsr"`
	T   string  `json:"t"`
	S   int64   `json:"s"`
}

// --- Derivative Models ---

// TotalCashDerivativeResponse represents derivative cash/margin overview.
type TotalCashDerivativeResponse struct {
	Fee             float64 `json:"fee"`
	Tax             float64 `json:"tax"`
	Others          float64 `json:"others"`
	CashWithdraw    float64 `json:"cashWithdraw"`
	TienBoSung      float64 `json:"tienbosung"`
	CashAvailWithdraw float64 `json:"cashavaiwithdraw"`
	Assets          float64 `json:"assets"`
	NAV             float64 `json:"nav"`
	CashOut         float64 `json:"cashOut"`
	VSDDeposit      float64 `json:"vsdDeposit"`
	IM              float64 `json:"im"`
	Cash            float64 `json:"cash"`
	PL              float64 `json:"pl"`
	VM              float64 `json:"vm"`
	EE              float64 `json:"ee"`
}

// AssetPositionCloseDerivativeResponse represents a closed derivative position.
type AssetPositionCloseDerivativeResponse struct {
	Symbol        string  `json:"symbol"`
	Side          string  `json:"side"`
	OpenPrice     float64 `json:"openPrice"`
	ClosePrice    float64 `json:"closePrice"`
	ClosePosition any     `json:"closePosition"`
	Fee           float64 `json:"fee"`
	Tax           float64 `json:"tax"`
	CloseVM       float64 `json:"closeVM"`
	Unrealize     float64 `json:"unrealize"`
	ClosePC       float64 `json:"closePC"`
	Time          string  `json:"time"`
}

// AssetPositionOpenDerivativeResponse represents an open derivative position.
type AssetPositionOpenDerivativeResponse struct {
	Symbol      string  `json:"symbol"`
	IM          string  `json:"im"`
	Deliver     string  `json:"deliver"`
	Receive     string  `json:"receive"`
	Net         float64 `json:"net"`
	Side        string  `json:"side"`
	Account     string  `json:"account"`
	WASP        float64 `json:"wasp"`
	WAPB        float64 `json:"wapb"`
	LastPrice   float64 `json:"lastPrice"`
	IMValue     float64 `json:"imValue"`
	VMValue     float64 `json:"vmValue"`
	MRValue     float64 `json:"mrValue"`
	DueDate     string  `json:"duedate"`
	NetOffVol   float64 `json:"netoffvol"`
	AvgRemain   float64 `json:"avg_remain"`
	VMRemain    float64 `json:"vm_remain"`
	PCRemain    string  `json:"pc_remain"`
	StopLoss    string  `json:"stoploss"`
	TakeProfit  string  `json:"takeprofit"`
}

// DerivativeNormalOrderResponse represents a normal derivative order.
type DerivativeNormalOrderResponse struct {
	OrderNo     string `json:"orderNo"`
	PKOrderNo   string `json:"pk_orderNo"`
	RefID       string `json:"refId"`
	OrderTime   string `json:"orderTime"`
	AccountCode string `json:"accountCode"`
	Side        string `json:"side"`
	Symbol      string `json:"symbol"`
	Volume      string `json:"volume"`
	ShowPrice   string `json:"showPrice"`
	MatchVolume string `json:"matchVolume"`
	Status      string `json:"status"`
	OrderStatus string `json:"orderStatus"`
	Channel     string `json:"channel"`
	Group       string `json:"group"`
}

// DerivativeConditionOrderResponse represents a conditional derivative order.
type DerivativeConditionOrderResponse struct {
	OrderNo     string `json:"orderNo"`
	RefID       string `json:"refId"`
	OrderTime   string `json:"orderTime"`
	AccountCode string `json:"accountCode"`
	Side        string `json:"side"`
	Symbol      string `json:"symbol"`
	Volume      string `json:"volume"`
	Price       string `json:"price"`
	Status      string `json:"status"`
	OrderType   string `json:"orderType"`
}

// DerivativeNormalOrderRequest represents a request to place a normal derivative order.
type DerivativeNormalOrderRequest struct {
	AccountID    string `json:"accountId"`
	SubAccountID string `json:"subAccountId"`
	Symbol       string `json:"symbol"`
	Side         string `json:"side"`
	OrderType    string `json:"orderType"`
	Volume       int    `json:"volume"`
	Price        string `json:"price"`
}

// DerivativeConditionOrderRequest represents a request to place a conditional derivative order.
type DerivativeConditionOrderRequest struct {
	AccountID    string `json:"accountId"`
	SubAccountID string `json:"subAccountId"`
	Symbol       string `json:"symbol"`
	Side         string `json:"side"`
	OrderType    string `json:"orderType"`
	Volume       int    `json:"volume"`
	Price        string `json:"price"`
	StopPrice    string `json:"stopPrice,omitempty"`
	TakeProfit   string `json:"takeProfit,omitempty"`
	StopLoss     string `json:"stopLoss,omitempty"`
}

// DerivativeChangeOrderRequest represents a request to modify a derivative order.
type DerivativeChangeOrderRequest struct {
	AccountID    string `json:"accountId"`
	SubAccountID string `json:"subAccountId"`
	RefID        string `json:"refId"`
	Volume       int    `json:"volume"`
	Price        string `json:"price"`
}

// DerivativeCancelOrderRequest represents a request to cancel a derivative order.
type DerivativeCancelOrderRequest struct {
	AccountID    string `json:"accountId"`
	SubAccountID string `json:"subAccountId"`
	RefID        string `json:"refId"`
}

// DerivativeMarketInfo represents derivative contract pricing and information.
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
	Cmd  string `json:"cmd"`
	RC   string `json:"rc"`
	RS   string `json:"rs"`
	OID  string `json:"oID"`
	Data string `json:"data"`
}
