package cryptopurchase

import (
	"math"
	"testing"
	"time"
)

func TestNewCryptoPurchase(t *testing.T) {
	t.Run("valid purchase with strategy", func(t *testing.T) {
		cp, err := NewCryptoPurchase(
			"tx-1", "acc-1", "profile-1", "SOL",
			0.02997, 66.75, 5.00,
			time.Date(2024, 12, 15, 0, 0, 0, 0, time.UTC),
			"compra SOL", "grid-bot-1",
		)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if cp.Asset != "SOL" {
			t.Errorf("Asset = %s, want SOL", cp.Asset)
		}
		if cp.Strategy != "grid-bot-1" {
			t.Errorf("Strategy = %s, want grid-bot-1", cp.Strategy)
		}
		if cp.Quantity != 0.02997 {
			t.Errorf("Quantity = %f, want 0.02997", cp.Quantity)
		}
		// InvestedUSD = 0.02997 * 66.75 = 2.000475
		expectedUSD := 0.02997 * 66.75
		if math.Abs(cp.InvestedUSD-expectedUSD) > 0.001 {
			t.Errorf("InvestedUSD = %f, want %f", cp.InvestedUSD, expectedUSD)
		}
		// InvestedBRL = investedUSD * 5.00
		expectedBRL := expectedUSD * 5.00
		if math.Abs(cp.InvestedBRL-expectedBRL) > 0.01 {
			t.Errorf("InvestedBRL = %f, want %f", cp.InvestedBRL, expectedBRL)
		}
	})

	t.Run("empty strategy defaults to manual", func(t *testing.T) {
		cp, err := NewCryptoPurchase("tx-1", "acc-1", "p-1", "SOL", 1, 100, 5, time.Now(), "", "")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if cp.Strategy != "manual" {
			t.Errorf("Strategy = %s, want manual", cp.Strategy)
		}
	})

	t.Run("empty asset", func(t *testing.T) {
		_, err := NewCryptoPurchase("tx-1", "acc-1", "p-1", "", 1, 100, 5, time.Now(), "", "")
		if err != ErrInvalidAsset {
			t.Errorf("err = %v, want ErrInvalidAsset", err)
		}
	})

	t.Run("zero quantity", func(t *testing.T) {
		_, err := NewCryptoPurchase("tx-1", "acc-1", "p-1", "SOL", 0, 100, 5, time.Now(), "", "")
		if err != ErrInvalidQuantity {
			t.Errorf("err = %v, want ErrInvalidQuantity", err)
		}
	})

	t.Run("negative price", func(t *testing.T) {
		_, err := NewCryptoPurchase("tx-1", "acc-1", "p-1", "SOL", 1, -10, 5, time.Now(), "", "")
		if err != ErrInvalidPrice {
			t.Errorf("err = %v, want ErrInvalidPrice", err)
		}
	})

	t.Run("zero exchange rate", func(t *testing.T) {
		_, err := NewCryptoPurchase("tx-1", "acc-1", "p-1", "SOL", 1, 100, 0, time.Now(), "", "")
		if err != ErrInvalidRate {
			t.Errorf("err = %v, want ErrInvalidRate", err)
		}
	})
}

func TestFindByStrategy(t *testing.T) {
	t.Run("strategy is stored and queryable", func(t *testing.T) {
		cp1, _ := NewCryptoPurchase("tx-1", "acc-1", "p-1", "SOL", 1, 100, 5, time.Now(), "", "grid-bot-1")
		cp2, _ := NewCryptoPurchase("tx-2", "acc-1", "p-1", "SOL", 2, 110, 5, time.Now(), "", "dca-btc")
		cp3, _ := NewCryptoPurchase("tx-3", "acc-1", "p-1", "SOL", 3, 120, 5, time.Now(), "", "grid-bot-1")

		all := []*CryptoPurchase{cp1, cp2, cp3}
		var gridBot1 []*CryptoPurchase
		for _, cp := range all {
			if cp.Strategy == "grid-bot-1" {
				gridBot1 = append(gridBot1, cp)
			}
		}
		if len(gridBot1) != 2 {
			t.Errorf("filtered by strategy = %d, want 2", len(gridBot1))
		}
	})
}

func TestCalculateGains(t *testing.T) {
	// Purchase: 0.02997 SOL @ $66.75/SOL, USD/BRL 5.00
	// InvestedUSD = 2.000475, InvestedBRL = 10.002375
	cp, _ := NewCryptoPurchase(
		"tx-1", "acc-1", "profile-1", "SOL",
		0.02997, 66.75, 5.00,
		time.Date(2024, 12, 15, 0, 0, 0, 0, time.UTC),
		"", "grid-bot-1",
	)

	t.Run("crypto price up and exchange rate up", func(t *testing.T) {
		// SOL now $83.78, USD/BRL now 5.26
		gains := cp.CalculateGains(83.78, 5.26)

		// CurrentValueUSD = 0.02997 * 83.78
		expectedValueUSD := 0.02997 * 83.78
		if math.Abs(gains.CurrentValueUSD-expectedValueUSD) > 0.001 {
			t.Errorf("CurrentValueUSD = %f, want %f", gains.CurrentValueUSD, expectedValueUSD)
		}

		// CurrentValueBRL = currentValueUSD * 5.26
		expectedValueBRL := expectedValueUSD * 5.26
		if math.Abs(gains.CurrentValueBRL-expectedValueBRL) > 0.01 {
			t.Errorf("CurrentValueBRL = %f, want %f", gains.CurrentValueBRL, expectedValueBRL)
		}

		// GainCryptoUSD = quantity * (currentPrice - purchasePrice)
		expectedGainCrypto := 0.02997 * (83.78 - 66.75)
		if math.Abs(gains.GainCryptoUSD-expectedGainCrypto) > 0.001 {
			t.Errorf("GainCryptoUSD = %f, want %f", gains.GainCryptoUSD, expectedGainCrypto)
		}

		// GainCryptoPercent = ((83.78 - 66.75) / 66.75) * 100 ≈ 25.51%
		if gains.GainCryptoPercent < 25 || gains.GainCryptoPercent > 26 {
			t.Errorf("GainCryptoPercent = %f, want ~25.5%%", gains.GainCryptoPercent)
		}

		// GainExchangeBRL = investedUSD * (5.26 - 5.00)
		expectedGainExchange := cp.InvestedUSD * (5.26 - 5.00)
		if math.Abs(gains.GainExchangeBRL-expectedGainExchange) > 0.01 {
			t.Errorf("GainExchangeBRL = %f, want %f", gains.GainExchangeBRL, expectedGainExchange)
		}

		// GainExchangePercent = ((5.26 - 5.00) / 5.00) * 100 = 5.2%
		if math.Abs(gains.GainExchangePercent-5.2) > 0.1 {
			t.Errorf("GainExchangePercent = %f, want ~5.2%%", gains.GainExchangePercent)
		}

		// Total gain in BRL should be positive
		if gains.GainTotalBRL <= 0 {
			t.Errorf("GainTotalBRL = %f, want positive", gains.GainTotalBRL)
		}

		// GainTotalBRL = currentValueBRL - investedBRL
		expectedTotalGain := expectedValueBRL - cp.InvestedBRL
		if math.Abs(gains.GainTotalBRL-expectedTotalGain) > 0.01 {
			t.Errorf("GainTotalBRL = %f, want %f", gains.GainTotalBRL, expectedTotalGain)
		}
	})

	t.Run("crypto price down but exchange rate up enough to profit in BRL", func(t *testing.T) {
		// SOL drops to $50, but USD/BRL goes to 8.00
		gains := cp.CalculateGains(50.0, 8.0)

		// Crypto loss in USD
		if gains.GainCryptoUSD >= 0 {
			t.Errorf("GainCryptoUSD = %f, want negative (price dropped)", gains.GainCryptoUSD)
		}

		// Exchange gain in BRL
		if gains.GainExchangeBRL <= 0 {
			t.Errorf("GainExchangeBRL = %f, want positive (BRL devalued)", gains.GainExchangeBRL)
		}

		// Total in BRL: 0.02997 * 50 * 8.0 = 11.988 vs invested ~10.00 → still profit
		if gains.GainTotalBRL <= 0 {
			t.Errorf("GainTotalBRL = %f, want positive despite crypto loss", gains.GainTotalBRL)
		}
	})

	t.Run("everything drops", func(t *testing.T) {
		// SOL drops to $30, USD/BRL drops to 4.00
		gains := cp.CalculateGains(30.0, 4.0)

		if gains.GainCryptoUSD >= 0 {
			t.Errorf("GainCryptoUSD should be negative")
		}
		if gains.GainExchangeBRL >= 0 {
			t.Errorf("GainExchangeBRL should be negative")
		}
		if gains.GainTotalBRL >= 0 {
			t.Errorf("GainTotalBRL should be negative")
		}
	})
}

func TestRemainingQuantity(t *testing.T) {
	cp, _ := NewCryptoPurchase("tx-1", "acc-1", "p-1", "SOL", 1.0, 100, 5, time.Now(), "", "")

	t.Run("no sales", func(t *testing.T) {
		if cp.RemainingQuantity() != 1.0 {
			t.Errorf("RemainingQuantity = %f, want 1.0", cp.RemainingQuantity())
		}
	})

	t.Run("partial sale", func(t *testing.T) {
		cp.SoldQuantity = 0.3
		if math.Abs(cp.RemainingQuantity()-0.7) > 0.0001 {
			t.Errorf("RemainingQuantity = %f, want 0.7", cp.RemainingQuantity())
		}
	})

	t.Run("fully sold", func(t *testing.T) {
		cp.SoldQuantity = 1.0
		if cp.RemainingQuantity() != 0.0 {
			t.Errorf("RemainingQuantity = %f, want 0.0", cp.RemainingQuantity())
		}
	})
}

func TestIsFullySold(t *testing.T) {
	cp, _ := NewCryptoPurchase("tx-1", "acc-1", "p-1", "SOL", 1.0, 100, 5, time.Now(), "", "")

	if cp.IsFullySold() {
		t.Error("should not be fully sold initially")
	}

	cp.SoldQuantity = 1.0
	if !cp.IsFullySold() {
		t.Error("should be fully sold when SoldQuantity == Quantity")
	}
}

func TestMarkSold(t *testing.T) {
	t.Run("valid partial sell", func(t *testing.T) {
		cp, _ := NewCryptoPurchase("tx-1", "acc-1", "p-1", "SOL", 1.0, 100, 5, time.Now(), "", "")
		err := cp.MarkSold(0.5)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if cp.SoldQuantity != 0.5 {
			t.Errorf("SoldQuantity = %f, want 0.5", cp.SoldQuantity)
		}
	})

	t.Run("accumulates sold quantity", func(t *testing.T) {
		cp, _ := NewCryptoPurchase("tx-1", "acc-1", "p-1", "SOL", 1.0, 100, 5, time.Now(), "", "")
		cp.MarkSold(0.3)
		cp.MarkSold(0.4)
		if math.Abs(cp.SoldQuantity-0.7) > 0.0001 {
			t.Errorf("SoldQuantity = %f, want 0.7", cp.SoldQuantity)
		}
	})

	t.Run("overflow returns error", func(t *testing.T) {
		cp, _ := NewCryptoPurchase("tx-1", "acc-1", "p-1", "SOL", 1.0, 100, 5, time.Now(), "", "")
		err := cp.MarkSold(1.5)
		if err != ErrSoldExceedsQuantity {
			t.Errorf("err = %v, want ErrSoldExceedsQuantity", err)
		}
		if cp.SoldQuantity != 0 {
			t.Errorf("SoldQuantity should not change on error, got %f", cp.SoldQuantity)
		}
	})

	t.Run("zero quantity error", func(t *testing.T) {
		cp, _ := NewCryptoPurchase("tx-1", "acc-1", "p-1", "SOL", 1.0, 100, 5, time.Now(), "", "")
		err := cp.MarkSold(0)
		if err != ErrInvalidQuantity {
			t.Errorf("err = %v, want ErrInvalidQuantity", err)
		}
	})
}

func TestCalculateGains_PartialSold(t *testing.T) {
	// Buy 1.0 SOL @ $100, rate 5.0 -> invested $100, R$500
	cp, _ := NewCryptoPurchase("tx-1", "acc-1", "p-1", "SOL", 1.0, 100, 5, time.Now(), "", "")
	cp.MarkSold(0.6) // sold 0.6, remaining 0.4

	gains := cp.CalculateGains(150, 5.0)

	// CurrentValueUSD should be based on remaining: 0.4 * 150 = 60
	expectedValueUSD := 0.4 * 150
	if math.Abs(gains.CurrentValueUSD-expectedValueUSD) > 0.001 {
		t.Errorf("CurrentValueUSD = %f, want %f", gains.CurrentValueUSD, expectedValueUSD)
	}

	// CurrentValueBRL = 60 * 5 = 300
	expectedValueBRL := expectedValueUSD * 5.0
	if math.Abs(gains.CurrentValueBRL-expectedValueBRL) > 0.01 {
		t.Errorf("CurrentValueBRL = %f, want %f", gains.CurrentValueBRL, expectedValueBRL)
	}
}
