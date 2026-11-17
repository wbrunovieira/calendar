package middleware

import "testing"

func TestCheckTokenConfigured(t *testing.T) {
	tests := []struct {
		name     string
		required string
		token    string
		wantErr  bool
	}{
		{"not required, no token", "", "", false},
		{"explicitly not required", "false", "", false},
		{"required with token", "true", "tok", false},
		{"required, empty token", "true", "", true},
		{"required, blank token", "true", "   ", true},
		{"flag read loosely", " TRUE ", "", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := CheckTokenConfigured(tt.required, tt.token)
			if (err != nil) != tt.wantErr {
				t.Fatalf("err = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
