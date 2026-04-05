package tcbs

// AccountInformationResponse represents sub-account information.
type AccountInformationResponse struct {
	BasicInfo       *BasicInfo       `json:"basicInfo,omitempty"`
	PersonalInfo    *PersonalInfo    `json:"personalInfo,omitempty"`
	BankAccounts    []BankAccount    `json:"bankAccounts,omitempty"`
	BankSubAccounts []BankSubAccount `json:"bankSubAccounts,omitempty"`
}

// BasicInfo holds basic account information.
type BasicInfo struct {
	TcbsID     string `json:"tcbsId"`
	Code105C   string `json:"code105C"`
	Status     string `json:"status"`
	Type       string `json:"type"`
	Depository bool   `json:"depository"`
}

// PersonalInfo holds personal information.
type PersonalInfo struct {
	FullName          string        `json:"fullName"`
	FullNameNoAccent  string        `json:"fullNameNoAccent"`
	FirstName         string        `json:"firstName"`
	LastName          string        `json:"lastName"`
	Email             string        `json:"email"`
	PhoneNumber       string        `json:"phoneNumber"`
	Gender            string        `json:"gender"`
	Birthday          string        `json:"birthday"`
	ContactAddress    string        `json:"contactAddress"`
	PermanentAddress  string        `json:"permanentAddress"`
	Nationality       string        `json:"nationality"`
	NationalityName   string        `json:"nationalityName"`
	TaxIDNumber       string        `json:"taxIdNumber"`
	Acronym           string        `json:"acronym"`
	CreatedDate       string        `json:"createdDate"`
	UpdatedDate       string        `json:"updatedDate"`
	FlowOpenAccount   string        `json:"flowOpenAccount"`
	AvatarURL         string        `json:"avatarUrl"`
	BusinessType      string        `json:"businessType"`
	PPBusinessType    string        `json:"ppBusinessType"`
	PPBusinessField   string        `json:"ppBusinessField"`
	PPBusinessTypeName  string      `json:"ppBusinessTypeName"`
	PPBusinessFieldName string      `json:"ppBusinessFieldName"`
	IdentityCard      *IdentityCard `json:"identityCard,omitempty"`
}

// IdentityCard holds identity document information.
type IdentityCard struct {
	Object     string `json:"object"`
	IDNumber   string `json:"idNumner"` // note: typo in spec
	IDPlace    string `json:"idPlace"`
	IDDate     string `json:"idDate"`
	ExpireDate string `json:"expireDate"`
	IDType     string `json:"idType"`
}

// BankSubAccount represents a sub-account linked to a bank.
type BankSubAccount struct {
	AccountNo       string `json:"accountNo"`
	AccountName     string `json:"accountName"`
	AccountType     string `json:"accountType"`
	AccountTypeName string `json:"accountTypeName"`
	Status          string `json:"status"`
	IsDefault       string `json:"isDefault"`
}

// BankAccount represents a linked bank account.
type BankAccount struct {
	AccountNo          string `json:"accountNo"`
	AccountName        string `json:"accountName"`
	AccountNameNoAccent string `json:"accountNameNoAccent"`
	BankCode           string `json:"bankCode"`
	BankName           string `json:"bankName"`
	BranchCode         string `json:"branchCode"`
	BankType           string `json:"bankType"`
	BankSys            string `json:"bankSys"`
	Authorized         string `json:"authorized"`
	BankAccountType    string `json:"bankAccountType"`
}
