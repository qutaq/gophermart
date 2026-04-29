package luhn

import "testing"

func TestValid(t *testing.T) {
	tests := []struct {
		number string
		want   bool
	}{
		{"4532015112830366", true},  // Visa test card
		{"79927398713", true},       // canonical example from Wikipedia
		{"79927398710", false},
		{"79927398711", false},
		{"79927398712", false},
		{"12345674", true},
		{"12345675", false},
		{"", false},
		{"abc", false},
		{"49927398716", true},
		{"1234567812345670", true},
		{"1234567812345678", false},
		{"0", true},
		{"00", true},
	}

	for _, tt := range tests {
		got := Valid(tt.number)
		if got != tt.want {
			t.Errorf("Valid(%q) = %v, want %v", tt.number, got, tt.want)
		}
	}
}
