package usecases

import "testing"

func TestParseStrategyFromClientOrderID(t *testing.T) {
	tests := []struct {
		name          string
		clientOrderID string
		wantStrategy  string
	}{
		{
			name:          "bot with pair and timestamp",
			clientOrderID: "MACross1-SOLBRL-1743271234567",
			wantStrategy:  "MACross1",
		},
		{
			name:          "different bot name",
			clientOrderID: "GridBot2-BTCBRL-1743271298000",
			wantStrategy:  "GridBot2",
		},
		{
			name:          "bot with hyphen in name",
			clientOrderID: "MA-Cross-ETHBRL-1743271234567",
			wantStrategy:  "MA-Cross",
		},
		{
			name:          "web order (no pattern)",
			clientOrderID: "web_abc123def456",
			wantStrategy:  "manual",
		},
		{
			name:          "empty clientOrderId",
			clientOrderID: "",
			wantStrategy:  "manual",
		},
		{
			name:          "android app order",
			clientOrderID: "android_abc123",
			wantStrategy:  "manual",
		},
		{
			name:          "ios app order",
			clientOrderID: "ios_abc123",
			wantStrategy:  "manual",
		},
		{
			name:          "only bot id without timestamp",
			clientOrderID: "MACross1-SOLBRL",
			wantStrategy:  "manual",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseStrategyFromClientOrderID(tt.clientOrderID)
			if got != tt.wantStrategy {
				t.Errorf("parseStrategyFromClientOrderID(%q) = %q, want %q", tt.clientOrderID, got, tt.wantStrategy)
			}
		})
	}
}
