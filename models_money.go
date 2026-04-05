package tcbs

// MoneyTransferRequest represents an internal money transfer request.
type MoneyTransferRequest struct {
	SourceAccountNumber      string  `json:"sourceAccountNumber"`
	DestinationAccountNumber string  `json:"destinationAccountNumber"`
	Amount                   float64 `json:"amount"`
	Description              float64 `json:"description"` // number type per spec
}

// MoneyTransferResponse represents the transfer response.
type MoneyTransferResponse struct {
	Status  string `json:"status"`
	Message string `json:"message"`
}

// MarginDepositRequest represents a margin deposit request for derivative accounts.
type MarginDepositRequest struct {
	AccountID      string  `json:"accountId"`
	SubAccountID   string  `json:"subAccountId"`
	Amount         float64 `json:"amount"`
	PaymentContent float64 `json:"paymentContent"`
}

// MarginWithdrawRequest represents a margin withdrawal request for derivative accounts.
type MarginWithdrawRequest struct {
	AccountID      string  `json:"accountId"`
	SubAccountID   string  `json:"subAccountId"`
	Amount         float64 `json:"amount"`
	PaymentContent float64 `json:"paymentContent"`
}

// MarginDepositResponse represents the deposit response.
type MarginDepositResponse struct {
	TransactionID string `json:"transactionId"`
}

// MarginWithdrawResponse represents the withdrawal response.
type MarginWithdrawResponse struct {
	TransactionID string `json:"transactionId"`
}
