package tcbs

// --- Stock Assets ---

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

// --- Cash Balance ---

// CashInvestmentResponse represents cash balance information.
type CashInvestmentResponse struct {
	Object     string           `json:"object"`
	TotalCount int              `json:"totalCount"`
	PageSize   int              `json:"pageSize"`
	PageIndex  int              `json:"pageIndex"`
	Data       []CashInvestment `json:"data"`
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
	CashDividend     float64  `json:"cashDevident"` // note: typo in spec
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

// --- Cash Statements ---

// TransHistCashStatementsResponse represents cash statement history.
type TransHistCashStatementsResponse struct {
	Response *TransHistCashStatementsData `json:"response"`
}

// TransHistCashStatementsData holds the paged data of cash statements.
type TransHistCashStatementsData struct {
	PageIndex      int                  `json:"pageIndex"`
	PageSize       int                  `json:"pageSize"`
	TotalCreditAmt int64                `json:"totalCreditAmount"`
	TotalDebitAmt  int64                `json:"totalDebitAmount"`
	TotalCount     int                  `json:"totalCount"`
	Data           []CashStatementEntry `json:"data"`
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
	TransactionDate string  `json:"transationDate"` // note: typo in spec
	CreditAmount    float64 `json:"creditAmount"`
}

// --- Margin Info ---

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

// --- Supplementary Loan Package ---

// SupplementaryLoanPackageResponse represents supplementary loan package info.
type SupplementaryLoanPackageResponse struct {
	MarginSureViews []MarginSureView `json:"marginSureViews"`
	TPlus           *TPlusData       `json:"tplus,omitempty"`
}

// MarginSureView represents a margin-sure insurance package.
type MarginSureView struct {
	ID              float64              `json:"id"`
	Name            string               `json:"name"`
	Code            string               `json:"code"`
	SubscriptionFee float64              `json:"subscriptionFee"`
	Status          string               `json:"status"`
	Proposals       []MarginSureProposal `json:"proposals"`
	Default         bool                 `json:"default"`
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
	FirstRate                                    float64       `json:"firstRate"`
	ID                                           float64       `json:"id"`
	Name                                         string        `json:"name"`
	Status                                       string        `json:"status"`
	UndueInterestType                            string        `json:"undueInterestType"`
	UndueLadderValue                             []TPlusLadder `json:"undueLadderValue"`
	OverdueInterest                              float64       `json:"overdueInterest"`
	ExtensionInterest                            float64       `json:"extensionInterest"`
	ExtensionInterestBeforeInterestSettlement    float64       `json:"extensionInterestBeforeInterestSettlement"`
	InterestCalculationBasis                     float64       `json:"interestCalculationBasis"`
	UndueFee                                     float64       `json:"undueFee"`
	OverdueFee                                   float64       `json:"overdueFee"`
	ExtensionFee                                 float64       `json:"extensionFee"`
	DebtCollectionFee                            float64       `json:"debtCollectionFee"`
	Description                                  string        `json:"description"`
	ValidFrom                                    string        `json:"validFrom"`
}

// TPlusLadder represents a ladder interest rate tier.
type TPlusLadder struct {
	ID        float64 `json:"id"`
	Rate      float64 `json:"rate"`
	StartDate float64 `json:"startDate"`
	DueDate   float64 `json:"dueDate"`
}

// --- Loans ---

// LoanResponse represents the loan list response.
type LoanResponse struct {
	Size    int        `json:"size"`
	Content []LoanItem `json:"content"`
}

// LoanItem represents a single loan.
type LoanItem struct {
	OpeningDate        string   `json:"openingDate"`
	DueDate            string   `json:"dueDate"`
	RenewTime          int      `json:"renewTime"`
	MaxRenewTime       int      `json:"maxRenewTime"`
	IsRenewable        bool     `json:"isRenewable"`
	ReasonList         []string `json:"reasonList"`
	Symbol             string   `json:"symbol"`
	ID                 float64  `json:"id"`
	AccountNo          string   `json:"accountNo"`
	Principal          float64  `json:"principal"`
	RemainingPrincipal float64  `json:"remainingPrincipal"`
	Interest           float64  `json:"interest"`
	Rate               float64  `json:"rate"`
	Status             string   `json:"status"`
	LoanDays           int      `json:"loanDays"`
	MrxLoanID          float64  `json:"mrxLoanId"`
	Fee                float64  `json:"fee"`
	UndueLoanFee       float64  `json:"undueLoanFee"`
	PricingPolicyType  string   `json:"pricingPolicyType"`
}
