package bankaccount

import (
	"testing"
	"time"
)

func TestNewBankAccount(t *testing.T) {
	tests := []struct {
		name           string
		profileID      string
		accountName    string
		accountType    AccountType
		initialBalance float64
		currency       string
		wantErr        bool
	}{
		{
			name:           "valid checking account",
			profileID:      "profile-123",
			accountName:    "Main Account",
			accountType:    AccountTypeChecking,
			initialBalance: 1000.00,
			currency:       "BRL",
			wantErr:        false,
		},
		{
			name:           "valid investment account",
			profileID:      "profile-123",
			accountName:    "CDB Investment",
			accountType:    AccountTypeInvestment,
			initialBalance: 5000.00,
			currency:       "BRL",
			wantErr:        false,
		},
		{
			name:           "missing profile ID",
			profileID:      "",
			accountName:    "Test Account",
			accountType:    AccountTypeChecking,
			initialBalance: 100.00,
			currency:       "BRL",
			wantErr:        true,
		},
		{
			name:           "missing name",
			profileID:      "profile-123",
			accountName:    "",
			accountType:    AccountTypeChecking,
			initialBalance: 100.00,
			currency:       "BRL",
			wantErr:        true,
		},
		{
			name:           "invalid account type",
			profileID:      "profile-123",
			accountName:    "Test Account",
			accountType:    AccountType("INVALID"),
			initialBalance: 100.00,
			currency:       "BRL",
			wantErr:        true,
		},
		{
			name:           "missing currency",
			profileID:      "profile-123",
			accountName:    "Test Account",
			accountType:    AccountTypeChecking,
			initialBalance: 100.00,
			currency:       "",
			wantErr:        true,
		},
		{
			name:           "valid USD currency",
			profileID:      "profile-123",
			accountName:    "Binance",
			accountType:    AccountTypeInvestment,
			initialBalance: 100.00,
			currency:       "USD",
			wantErr:        false,
		},
		{
			name:           "valid EUR currency",
			profileID:      "profile-123",
			accountName:    "Euro Account",
			accountType:    AccountTypeChecking,
			initialBalance: 50.00,
			currency:       "EUR",
			wantErr:        false,
		},
		{
			name:           "invalid currency",
			profileID:      "profile-123",
			accountName:    "Test Account",
			accountType:    AccountTypeChecking,
			initialBalance: 100.00,
			currency:       "GBP",
			wantErr:        true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			account, err := NewBankAccount(tt.profileID, tt.accountName, tt.accountType, tt.initialBalance, tt.currency)
			if tt.wantErr {
				if err == nil {
					t.Errorf("NewBankAccount() expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Errorf("NewBankAccount() unexpected error: %v", err)
				return
			}
			if account.ID == "" {
				t.Error("NewBankAccount() ID should not be empty")
			}
			if account.ProfileID != tt.profileID {
				t.Errorf("NewBankAccount() ProfileID = %v, want %v", account.ProfileID, tt.profileID)
			}
			if account.Name != tt.accountName {
				t.Errorf("NewBankAccount() Name = %v, want %v", account.Name, tt.accountName)
			}
			if account.Type != tt.accountType {
				t.Errorf("NewBankAccount() Type = %v, want %v", account.Type, tt.accountType)
			}
			if account.InitialBalance != tt.initialBalance {
				t.Errorf("NewBankAccount() InitialBalance = %v, want %v", account.InitialBalance, tt.initialBalance)
			}
			if account.CurrentBalance != tt.initialBalance {
				t.Errorf("NewBankAccount() CurrentBalance = %v, want %v", account.CurrentBalance, tt.initialBalance)
			}
			if !account.IsActive {
				t.Error("NewBankAccount() IsActive should be true")
			}
		})
	}
}

func TestIsValidInvestmentType(t *testing.T) {
	validTypes := []InvestmentType{
		InvestmentTypeSavingsBox,
		InvestmentTypeCDB,
		InvestmentTypeLCI,
		InvestmentTypeLCA,
		InvestmentTypeStocks,
		InvestmentTypeFunds,
		InvestmentTypeFII,
		InvestmentTypeCrypto,
		InvestmentTypeTreasury,
		InvestmentTypeOther,
	}

	for _, invType := range validTypes {
		t.Run(string(invType), func(t *testing.T) {
			if !IsValidInvestmentType(invType) {
				t.Errorf("IsValidInvestmentType(%v) = false, want true", invType)
			}
		})
	}

	t.Run("invalid type", func(t *testing.T) {
		if IsValidInvestmentType(InvestmentType("INVALID")) {
			t.Error("IsValidInvestmentType(INVALID) = true, want false")
		}
	})
}

func TestIsValidYieldType(t *testing.T) {
	validTypes := []YieldType{
		YieldTypeFixed,
		YieldTypeCDIPercentage,
		YieldTypeIPCAPlus,
		YieldTypeVariable,
	}

	for _, yieldType := range validTypes {
		t.Run(string(yieldType), func(t *testing.T) {
			if !IsValidYieldType(yieldType) {
				t.Errorf("IsValidYieldType(%v) = false, want true", yieldType)
			}
		})
	}

	t.Run("invalid type", func(t *testing.T) {
		if IsValidYieldType(YieldType("INVALID")) {
			t.Error("IsValidYieldType(INVALID) = true, want false")
		}
	})
}

func TestBankAccount_IsInvestment(t *testing.T) {
	tests := []struct {
		accountType AccountType
		want        bool
	}{
		{AccountTypeInvestment, true},
		{AccountTypeChecking, false},
		{AccountTypeSavings, false},
		{AccountTypeCreditCard, false},
	}

	for _, tt := range tests {
		t.Run(string(tt.accountType), func(t *testing.T) {
			account, _ := NewBankAccount("profile-123", "Test", tt.accountType, 1000, "BRL")
			if got := account.IsInvestment(); got != tt.want {
				t.Errorf("IsInvestment() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestBankAccount_IsCreditCard(t *testing.T) {
	tests := []struct {
		accountType AccountType
		want        bool
	}{
		{AccountTypeCreditCard, true},
		{AccountTypeChecking, false},
		{AccountTypeInvestment, false},
	}

	for _, tt := range tests {
		t.Run(string(tt.accountType), func(t *testing.T) {
			account, _ := NewBankAccount("profile-123", "Test", tt.accountType, 0, "BRL")
			if got := account.IsCreditCard(); got != tt.want {
				t.Errorf("IsCreditCard() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestBankAccount_SetInvestmentDetails(t *testing.T) {
	invType := InvestmentTypeCDB
	yieldType := YieldTypeCDIPercentage
	yieldRate := float64(100)
	maturity := time.Now().AddDate(1, 0, 0) // 1 year from now
	broker := "Nubank"

	t.Run("success on investment account", func(t *testing.T) {
		account, _ := NewBankAccount("profile-123", "CDB Test", AccountTypeInvestment, 5000, "BRL")
		err := account.SetInvestmentDetails(&invType, &yieldType, &yieldRate, &maturity, &broker)
		if err != nil {
			t.Errorf("SetInvestmentDetails() unexpected error: %v", err)
		}
		if account.InvestmentType == nil || *account.InvestmentType != invType {
			t.Errorf("InvestmentType = %v, want %v", account.InvestmentType, invType)
		}
		if account.YieldType == nil || *account.YieldType != yieldType {
			t.Errorf("YieldType = %v, want %v", account.YieldType, yieldType)
		}
		if account.YieldRate == nil || *account.YieldRate != yieldRate {
			t.Errorf("YieldRate = %v, want %v", account.YieldRate, yieldRate)
		}
		if account.Broker == nil || *account.Broker != broker {
			t.Errorf("Broker = %v, want %v", account.Broker, broker)
		}
	})

	t.Run("fail on non-investment account", func(t *testing.T) {
		account, _ := NewBankAccount("profile-123", "Checking", AccountTypeChecking, 1000, "BRL")
		err := account.SetInvestmentDetails(&invType, &yieldType, &yieldRate, &maturity, &broker)
		if err == nil {
			t.Error("SetInvestmentDetails() expected error for non-investment account")
		}
	})

	t.Run("fail on invalid investment type", func(t *testing.T) {
		account, _ := NewBankAccount("profile-123", "Investment", AccountTypeInvestment, 1000, "BRL")
		invalidType := InvestmentType("INVALID")
		err := account.SetInvestmentDetails(&invalidType, &yieldType, &yieldRate, &maturity, &broker)
		if err == nil {
			t.Error("SetInvestmentDetails() expected error for invalid investment type")
		}
	})

	t.Run("fail on invalid yield type", func(t *testing.T) {
		account, _ := NewBankAccount("profile-123", "Investment", AccountTypeInvestment, 1000, "BRL")
		invalidYield := YieldType("INVALID")
		err := account.SetInvestmentDetails(&invType, &invalidYield, &yieldRate, &maturity, &broker)
		if err == nil {
			t.Error("SetInvestmentDetails() expected error for invalid yield type")
		}
	})
}

func TestBankAccount_GetYieldDescription(t *testing.T) {
	tests := []struct {
		name      string
		yieldType *YieldType
		yieldRate *float64
		want      string
	}{
		{
			name:      "fixed rate",
			yieldType: ptr(YieldTypeFixed),
			yieldRate: ptr(12.5),
			want:      "12.50% a.a.",
		},
		{
			name:      "CDI percentage",
			yieldType: ptr(YieldTypeCDIPercentage),
			yieldRate: ptr(float64(100)),
			want:      "100% do CDI",
		},
		{
			name:      "IPCA plus",
			yieldType: ptr(YieldTypeIPCAPlus),
			yieldRate: ptr(5.5),
			want:      "IPCA + 5.50%",
		},
		{
			name:      "variable",
			yieldType: ptr(YieldTypeVariable),
			yieldRate: nil,
			want:      "Variável",
		},
		{
			name:      "no yield type",
			yieldType: nil,
			yieldRate: nil,
			want:      "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			account, _ := NewBankAccount("profile-123", "Investment", AccountTypeInvestment, 1000, "BRL")
			account.YieldType = tt.yieldType
			account.YieldRate = tt.yieldRate

			if got := account.GetYieldDescription(); got != tt.want {
				t.Errorf("GetYieldDescription() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestBankAccount_HasFixedMaturity(t *testing.T) {
	t.Run("with maturity date", func(t *testing.T) {
		account, _ := NewBankAccount("profile-123", "CDB", AccountTypeInvestment, 1000, "BRL")
		maturity := time.Now().AddDate(1, 0, 0)
		account.MaturityDate = &maturity

		if !account.HasFixedMaturity() {
			t.Error("HasFixedMaturity() = false, want true")
		}
	})

	t.Run("without maturity date", func(t *testing.T) {
		account, _ := NewBankAccount("profile-123", "Stocks", AccountTypeInvestment, 1000, "BRL")

		if account.HasFixedMaturity() {
			t.Error("HasFixedMaturity() = true, want false")
		}
	})
}

func TestBankAccount_IsMatured(t *testing.T) {
	t.Run("matured investment", func(t *testing.T) {
		account, _ := NewBankAccount("profile-123", "CDB", AccountTypeInvestment, 1000, "BRL")
		pastDate := time.Now().AddDate(0, 0, -1) // yesterday
		account.MaturityDate = &pastDate

		if !account.IsMatured() {
			t.Error("IsMatured() = false, want true")
		}
	})

	t.Run("not matured investment", func(t *testing.T) {
		account, _ := NewBankAccount("profile-123", "CDB", AccountTypeInvestment, 1000, "BRL")
		futureDate := time.Now().AddDate(1, 0, 0) // 1 year from now
		account.MaturityDate = &futureDate

		if account.IsMatured() {
			t.Error("IsMatured() = true, want false")
		}
	})

	t.Run("no maturity date", func(t *testing.T) {
		account, _ := NewBankAccount("profile-123", "Stocks", AccountTypeInvestment, 1000, "BRL")

		if account.IsMatured() {
			t.Error("IsMatured() = true, want false for no maturity")
		}
	})
}

func TestBankAccount_DaysToMaturity(t *testing.T) {
	t.Run("future maturity", func(t *testing.T) {
		account, _ := NewBankAccount("profile-123", "CDB", AccountTypeInvestment, 1000, "BRL")
		futureDate := time.Now().AddDate(0, 0, 30) // 30 days from now
		account.MaturityDate = &futureDate

		days := account.DaysToMaturity()
		// Allow 1 day variance for timing issues
		if days < 29 || days > 31 {
			t.Errorf("DaysToMaturity() = %d, want ~30", days)
		}
	})

	t.Run("past maturity", func(t *testing.T) {
		account, _ := NewBankAccount("profile-123", "CDB", AccountTypeInvestment, 1000, "BRL")
		pastDate := time.Now().AddDate(0, 0, -10)
		account.MaturityDate = &pastDate

		if days := account.DaysToMaturity(); days != 0 {
			t.Errorf("DaysToMaturity() = %d, want 0 for past date", days)
		}
	})

	t.Run("no maturity", func(t *testing.T) {
		account, _ := NewBankAccount("profile-123", "Stocks", AccountTypeInvestment, 1000, "BRL")

		if days := account.DaysToMaturity(); days != -1 {
			t.Errorf("DaysToMaturity() = %d, want -1 for no maturity", days)
		}
	})
}

func TestBankAccount_CalculateReturn(t *testing.T) {
	tests := []struct {
		name           string
		initialBalance float64
		currentBalance float64
		wantReturn     float64
	}{
		{
			name:           "positive return",
			initialBalance: 1000.00,
			currentBalance: 1100.00,
			wantReturn:     100.00,
		},
		{
			name:           "negative return",
			initialBalance: 1000.00,
			currentBalance: 900.00,
			wantReturn:     -100.00,
		},
		{
			name:           "no return",
			initialBalance: 1000.00,
			currentBalance: 1000.00,
			wantReturn:     0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			account, _ := NewBankAccount("profile-123", "Investment", AccountTypeInvestment, tt.initialBalance, "BRL")
			account.CurrentBalance = tt.currentBalance

			if got := account.CalculateReturn(); got != tt.wantReturn {
				t.Errorf("CalculateReturn() = %v, want %v", got, tt.wantReturn)
			}
		})
	}
}

func TestBankAccount_CalculateReturnPercentage(t *testing.T) {
	tests := []struct {
		name           string
		initialBalance float64
		currentBalance float64
		wantPercent    float64
	}{
		{
			name:           "10% return",
			initialBalance: 1000.00,
			currentBalance: 1100.00,
			wantPercent:    10.00,
		},
		{
			name:           "10% loss",
			initialBalance: 1000.00,
			currentBalance: 900.00,
			wantPercent:    -10.00,
		},
		{
			name:           "zero initial balance",
			initialBalance: 0,
			currentBalance: 100.00,
			wantPercent:    0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			account, _ := NewBankAccount("profile-123", "Investment", AccountTypeInvestment, tt.initialBalance, "BRL")
			account.CurrentBalance = tt.currentBalance

			if got := account.CalculateReturnPercentage(); got != tt.wantPercent {
				t.Errorf("CalculateReturnPercentage() = %v, want %v", got, tt.wantPercent)
			}
		})
	}
}

func TestBankAccount_UpdateBalance(t *testing.T) {
	account, _ := NewBankAccount("profile-123", "Test", AccountTypeChecking, 1000, "BRL")
	oldUpdatedAt := account.UpdatedAt

	time.Sleep(time.Millisecond) // ensure time difference
	account.UpdateBalance(1500)

	if account.CurrentBalance != 1500 {
		t.Errorf("UpdateBalance() CurrentBalance = %v, want 1500", account.CurrentBalance)
	}
	if !account.UpdatedAt.After(oldUpdatedAt) {
		t.Error("UpdateBalance() should update UpdatedAt")
	}
}

func TestBankAccount_ActivateDeactivate(t *testing.T) {
	account, _ := NewBankAccount("profile-123", "Test", AccountTypeChecking, 1000, "BRL")

	account.Deactivate()
	if account.IsActive {
		t.Error("Deactivate() IsActive should be false")
	}

	account.Activate()
	if !account.IsActive {
		t.Error("Activate() IsActive should be true")
	}
}

func TestBankAccount_CanLinkToAccount(t *testing.T) {
	checkingAccount, _ := NewBankAccount("profile-123", "Checking", AccountTypeChecking, 1000, "BRL")
	savingsAccount, _ := NewBankAccount("profile-123", "Savings", AccountTypeSavings, 500, "BRL")
	investmentAccount, _ := NewBankAccount("profile-123", "CDB", AccountTypeInvestment, 5000, "BRL")
	creditCardAccount, _ := NewBankAccount("profile-123", "Credit Card", AccountTypeCreditCard, 0, "BRL")
	otherProfileAccount, _ := NewBankAccount("profile-456", "Other Checking", AccountTypeChecking, 1000, "BRL")

	tests := []struct {
		name          string
		account       *BankAccount
		targetAccount *BankAccount
		want          bool
	}{
		{
			name:          "investment can link to checking",
			account:       investmentAccount,
			targetAccount: checkingAccount,
			want:          true,
		},
		{
			name:          "investment can link to savings",
			account:       investmentAccount,
			targetAccount: savingsAccount,
			want:          true,
		},
		{
			name:          "credit card can link to checking",
			account:       creditCardAccount,
			targetAccount: checkingAccount,
			want:          true,
		},
		{
			name:          "checking cannot link to another account",
			account:       checkingAccount,
			targetAccount: savingsAccount,
			want:          false,
		},
		{
			name:          "investment cannot link to credit card",
			account:       investmentAccount,
			targetAccount: creditCardAccount,
			want:          false,
		},
		{
			name:          "investment cannot link to another investment",
			account:       investmentAccount,
			targetAccount: investmentAccount,
			want:          false,
		},
		{
			name:          "cannot link to account from different profile",
			account:       investmentAccount,
			targetAccount: otherProfileAccount,
			want:          false,
		},
		{
			name:          "cannot link to nil account",
			account:       investmentAccount,
			targetAccount: nil,
			want:          false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.account.CanLinkToAccount(tt.targetAccount); got != tt.want {
				t.Errorf("CanLinkToAccount() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestBankAccount_SetLinkedAccount(t *testing.T) {
	t.Run("investment successfully links to checking", func(t *testing.T) {
		investment, _ := NewBankAccount("profile-123", "CDB", AccountTypeInvestment, 5000, "BRL")
		checking, _ := NewBankAccount("profile-123", "Checking", AccountTypeChecking, 1000, "BRL")

		err := investment.SetLinkedAccount(checking)
		if err != nil {
			t.Errorf("SetLinkedAccount() unexpected error: %v", err)
		}
		if investment.LinkedAccountID == nil || *investment.LinkedAccountID != checking.ID {
			t.Errorf("LinkedAccountID = %v, want %v", investment.LinkedAccountID, checking.ID)
		}
	})

	t.Run("credit card successfully links to checking", func(t *testing.T) {
		creditCard, _ := NewBankAccount("profile-123", "Card", AccountTypeCreditCard, 0, "BRL")
		checking, _ := NewBankAccount("profile-123", "Checking", AccountTypeChecking, 1000, "BRL")

		err := creditCard.SetLinkedAccount(checking)
		if err != nil {
			t.Errorf("SetLinkedAccount() unexpected error: %v", err)
		}
		if creditCard.LinkedAccountID == nil || *creditCard.LinkedAccountID != checking.ID {
			t.Errorf("LinkedAccountID = %v, want %v", creditCard.LinkedAccountID, checking.ID)
		}
	})

	t.Run("checking account cannot link", func(t *testing.T) {
		checking, _ := NewBankAccount("profile-123", "Checking", AccountTypeChecking, 1000, "BRL")
		savings, _ := NewBankAccount("profile-123", "Savings", AccountTypeSavings, 500, "BRL")

		err := checking.SetLinkedAccount(savings)
		if err == nil {
			t.Error("SetLinkedAccount() expected error for checking account")
		}
	})

	t.Run("cannot link to different profile", func(t *testing.T) {
		investment, _ := NewBankAccount("profile-123", "CDB", AccountTypeInvestment, 5000, "BRL")
		otherChecking, _ := NewBankAccount("profile-456", "Checking", AccountTypeChecking, 1000, "BRL")

		err := investment.SetLinkedAccount(otherChecking)
		if err == nil {
			t.Error("SetLinkedAccount() expected error for different profile")
		}
	})

	t.Run("cannot link to investment", func(t *testing.T) {
		investment1, _ := NewBankAccount("profile-123", "CDB 1", AccountTypeInvestment, 5000, "BRL")
		investment2, _ := NewBankAccount("profile-123", "CDB 2", AccountTypeInvestment, 3000, "BRL")

		err := investment1.SetLinkedAccount(investment2)
		if err == nil {
			t.Error("SetLinkedAccount() expected error for linking to investment")
		}
	})

	t.Run("cannot link to nil account", func(t *testing.T) {
		investment, _ := NewBankAccount("profile-123", "CDB", AccountTypeInvestment, 5000, "BRL")

		err := investment.SetLinkedAccount(nil)
		if err == nil {
			t.Error("SetLinkedAccount() expected error for nil account")
		}
	})
}

func TestBankAccount_ClearLinkedAccount(t *testing.T) {
	investment, _ := NewBankAccount("profile-123", "CDB", AccountTypeInvestment, 5000, "BRL")
	checking, _ := NewBankAccount("profile-123", "Checking", AccountTypeChecking, 1000, "BRL")

	// Link first
	_ = investment.SetLinkedAccount(checking)
	if investment.LinkedAccountID == nil {
		t.Fatal("LinkedAccountID should not be nil after SetLinkedAccount")
	}

	// Clear
	investment.ClearLinkedAccount()
	if investment.LinkedAccountID != nil {
		t.Error("ClearLinkedAccount() LinkedAccountID should be nil")
	}
}

func TestBankAccount_IsLinked(t *testing.T) {
	t.Run("linked account", func(t *testing.T) {
		investment, _ := NewBankAccount("profile-123", "CDB", AccountTypeInvestment, 5000, "BRL")
		checking, _ := NewBankAccount("profile-123", "Checking", AccountTypeChecking, 1000, "BRL")
		_ = investment.SetLinkedAccount(checking)

		if !investment.IsLinked() {
			t.Error("IsLinked() = false, want true")
		}
	})

	t.Run("unlinked account", func(t *testing.T) {
		investment, _ := NewBankAccount("profile-123", "CDB", AccountTypeInvestment, 5000, "BRL")

		if investment.IsLinked() {
			t.Error("IsLinked() = true, want false")
		}
	})
}

func TestBankAccount_RequiresLinking(t *testing.T) {
	tests := []struct {
		accountType AccountType
		want        bool
	}{
		{AccountTypeInvestment, true},
		{AccountTypeCreditCard, true},
		{AccountTypeChecking, false},
		{AccountTypeSavings, false},
		{AccountTypeCash, false},
		{AccountTypeOther, false},
	}

	for _, tt := range tests {
		t.Run(string(tt.accountType), func(t *testing.T) {
			account, _ := NewBankAccount("profile-123", "Test", tt.accountType, 1000, "BRL")
			if got := account.RequiresLinking(); got != tt.want {
				t.Errorf("RequiresLinking() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestBankAccount_IsValidLinkTarget(t *testing.T) {
	tests := []struct {
		accountType AccountType
		want        bool
	}{
		{AccountTypeChecking, true},
		{AccountTypeSavings, true},
		{AccountTypeCash, true},
		{AccountTypeInvestment, false},
		{AccountTypeCreditCard, false},
		{AccountTypeOther, false},
	}

	for _, tt := range tests {
		t.Run(string(tt.accountType), func(t *testing.T) {
			account, _ := NewBankAccount("profile-123", "Test", tt.accountType, 1000, "BRL")
			if got := account.IsValidLinkTarget(); got != tt.want {
				t.Errorf("IsValidLinkTarget() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestBankAccount_SetQuotasFromTotal(t *testing.T) {
	t.Run("calculate quota price from total and number of quotas", func(t *testing.T) {
		account, _ := NewBankAccount("profile-123", "FII XPLG11", AccountTypeInvestment, 0, "BRL")
		invType := InvestmentTypeFII
		account.InvestmentType = &invType

		// Total R$ 1000, 10 quotas = R$ 100 per quota
		err := account.SetQuotasFromTotal(10, 1000.00)
		if err != nil {
			t.Errorf("SetQuotasFromTotal() unexpected error: %v", err)
		}
		if account.NumberOfQuotas == nil || *account.NumberOfQuotas != 10 {
			t.Errorf("NumberOfQuotas = %v, want 10", account.NumberOfQuotas)
		}
		if account.QuotaPrice == nil || *account.QuotaPrice != 100.00 {
			t.Errorf("QuotaPrice = %v, want 100.00", account.QuotaPrice)
		}
		if account.InitialBalance != 1000.00 {
			t.Errorf("InitialBalance = %v, want 1000.00", account.InitialBalance)
		}
	})

	t.Run("fail on non-investment account", func(t *testing.T) {
		account, _ := NewBankAccount("profile-123", "Checking", AccountTypeChecking, 1000, "BRL")
		err := account.SetQuotasFromTotal(10, 1000.00)
		if err == nil {
			t.Error("SetQuotasFromTotal() expected error for non-investment account")
		}
	})

	t.Run("fail on zero quotas", func(t *testing.T) {
		account, _ := NewBankAccount("profile-123", "FII", AccountTypeInvestment, 0, "BRL")
		err := account.SetQuotasFromTotal(0, 1000.00)
		if err == nil {
			t.Error("SetQuotasFromTotal() expected error for zero quotas")
		}
	})

	t.Run("fail on negative total", func(t *testing.T) {
		account, _ := NewBankAccount("profile-123", "FII", AccountTypeInvestment, 0, "BRL")
		err := account.SetQuotasFromTotal(10, -100.00)
		if err == nil {
			t.Error("SetQuotasFromTotal() expected error for negative total")
		}
	})
}

func TestBankAccount_SetQuotasFromPrice(t *testing.T) {
	t.Run("calculate total from quota price and number of quotas", func(t *testing.T) {
		account, _ := NewBankAccount("profile-123", "PETR4", AccountTypeInvestment, 0, "BRL")
		invType := InvestmentTypeStocks
		account.InvestmentType = &invType

		// 50 shares at R$ 35.50 each = R$ 1775.00 total
		err := account.SetQuotasFromPrice(50, 35.50)
		if err != nil {
			t.Errorf("SetQuotasFromPrice() unexpected error: %v", err)
		}
		if account.NumberOfQuotas == nil || *account.NumberOfQuotas != 50 {
			t.Errorf("NumberOfQuotas = %v, want 50", account.NumberOfQuotas)
		}
		if account.QuotaPrice == nil || *account.QuotaPrice != 35.50 {
			t.Errorf("QuotaPrice = %v, want 35.50", account.QuotaPrice)
		}
		if account.InitialBalance != 1775.00 {
			t.Errorf("InitialBalance = %v, want 1775.00", account.InitialBalance)
		}
		if account.CurrentBalance != 1775.00 {
			t.Errorf("CurrentBalance = %v, want 1775.00", account.CurrentBalance)
		}
	})

	t.Run("fail on non-investment account", func(t *testing.T) {
		account, _ := NewBankAccount("profile-123", "Checking", AccountTypeChecking, 1000, "BRL")
		err := account.SetQuotasFromPrice(10, 100.00)
		if err == nil {
			t.Error("SetQuotasFromPrice() expected error for non-investment account")
		}
	})

	t.Run("fail on zero quotas", func(t *testing.T) {
		account, _ := NewBankAccount("profile-123", "Stocks", AccountTypeInvestment, 0, "BRL")
		err := account.SetQuotasFromPrice(0, 100.00)
		if err == nil {
			t.Error("SetQuotasFromPrice() expected error for zero quotas")
		}
	})

	t.Run("fail on negative price", func(t *testing.T) {
		account, _ := NewBankAccount("profile-123", "Stocks", AccountTypeInvestment, 0, "BRL")
		err := account.SetQuotasFromPrice(10, -50.00)
		if err == nil {
			t.Error("SetQuotasFromPrice() expected error for negative price")
		}
	})
}

func TestBankAccount_HasQuotas(t *testing.T) {
	t.Run("with quotas", func(t *testing.T) {
		account, _ := NewBankAccount("profile-123", "FII", AccountTypeInvestment, 1000, "BRL")
		_ = account.SetQuotasFromTotal(10, 1000.00)

		if !account.HasQuotas() {
			t.Error("HasQuotas() = false, want true")
		}
	})

	t.Run("without quotas", func(t *testing.T) {
		account, _ := NewBankAccount("profile-123", "CDB", AccountTypeInvestment, 1000, "BRL")

		if account.HasQuotas() {
			t.Error("HasQuotas() = true, want false")
		}
	})
}

func TestBankAccount_UpdateQuotaPrice(t *testing.T) {
	t.Run("update quota price and recalculate current balance", func(t *testing.T) {
		account, _ := NewBankAccount("profile-123", "XPLG11", AccountTypeInvestment, 0, "BRL")
		invType := InvestmentTypeFII
		account.InvestmentType = &invType

		// Initial: 10 quotas at R$ 100 = R$ 1000
		_ = account.SetQuotasFromPrice(10, 100.00)

		// Price goes up to R$ 110 per quota
		err := account.UpdateQuotaPrice(110.00)
		if err != nil {
			t.Errorf("UpdateQuotaPrice() unexpected error: %v", err)
		}
		if account.QuotaPrice == nil || *account.QuotaPrice != 110.00 {
			t.Errorf("QuotaPrice = %v, want 110.00", account.QuotaPrice)
		}
		// New balance should be 10 * 110 = 1100
		if account.CurrentBalance != 1100.00 {
			t.Errorf("CurrentBalance = %v, want 1100.00", account.CurrentBalance)
		}
		// Initial balance should remain unchanged
		if account.InitialBalance != 1000.00 {
			t.Errorf("InitialBalance = %v, want 1000.00", account.InitialBalance)
		}
	})

	t.Run("fail on account without quotas", func(t *testing.T) {
		account, _ := NewBankAccount("profile-123", "CDB", AccountTypeInvestment, 1000, "BRL")
		err := account.UpdateQuotaPrice(100.00)
		if err == nil {
			t.Error("UpdateQuotaPrice() expected error for account without quotas")
		}
	})

	t.Run("fail on negative price", func(t *testing.T) {
		account, _ := NewBankAccount("profile-123", "FII", AccountTypeInvestment, 0, "BRL")
		_ = account.SetQuotasFromPrice(10, 100.00)
		err := account.UpdateQuotaPrice(-50.00)
		if err == nil {
			t.Error("UpdateQuotaPrice() expected error for negative price")
		}
	})
}

func TestBankAccount_SupportsQuotas(t *testing.T) {
	tests := []struct {
		investmentType InvestmentType
		want           bool
	}{
		{InvestmentTypeStocks, true},
		{InvestmentTypeFII, true},
		{InvestmentTypeFunds, true},
		{InvestmentTypeCrypto, true},
		{InvestmentTypeCDB, false},
		{InvestmentTypeLCI, false},
		{InvestmentTypeLCA, false},
		{InvestmentTypeTreasury, false},
		{InvestmentTypeSavingsBox, false},
		{InvestmentTypeOther, false},
	}

	for _, tt := range tests {
		t.Run(string(tt.investmentType), func(t *testing.T) {
			account, _ := NewBankAccount("profile-123", "Test", AccountTypeInvestment, 1000, "BRL")
			account.InvestmentType = &tt.investmentType
			if got := account.SupportsQuotas(); got != tt.want {
				t.Errorf("SupportsQuotas() = %v, want %v", got, tt.want)
			}
		})
	}

	t.Run("nil investment type", func(t *testing.T) {
		account, _ := NewBankAccount("profile-123", "Test", AccountTypeInvestment, 1000, "BRL")
		if account.SupportsQuotas() {
			t.Error("SupportsQuotas() = true, want false for nil investment type")
		}
	})
}

// Helper function to create pointers
func ptr[T any](v T) *T {
	return &v
}
