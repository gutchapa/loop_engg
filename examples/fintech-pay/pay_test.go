package pay

import (
	"testing"
)

func TestLuhnValid(t *testing.T) {
	cases := []string{
		"4111111111111111", // Visa
		"5500000000000004", // Mastercard
		"378282246310005",  // Amex
		"4000056655665556",
	}
	for _, c := range cases {
		if !LuhnCheck(c) {
			t.Errorf("expected valid: %s", c)
		}
	}
}

func TestLuhnInvalid(t *testing.T) {
	cases := []string{
		"1234567890123456",
		"0000000000000000",
		"4111111111111112",
		"",
		"abc",
	}
	for _, c := range cases {
		if LuhnCheck(c) {
			t.Errorf("expected invalid: %s", c)
		}
	}
}

func TestValidateApprovesValid(t *testing.T) {
	tx := Transaction{
		CardNum: "4111111111111111",
		Amount:  50000,
	}
	if err := Validate(tx); err != nil {
		t.Errorf("expected valid, got: %v", err)
	}
}

func TestValidateRejectsOverLimit(t *testing.T) {
	tx := Transaction{
		CardNum: "4111111111111111",
		Amount:  1000000,
	}
	if err := Validate(tx); err == nil {
		t.Error("expected error for over-limit")
	}
}

func TestValidateRejectsNegativeAmount(t *testing.T) {
	tx := Transaction{
		CardNum: "4111111111111111",
		Amount:  -100,
	}
	if err := Validate(tx); err == nil {
		t.Error("expected error for negative amount")
	}
}

func TestValidateRejectsBadCard(t *testing.T) {
	tx := Transaction{
		CardNum: "0000000000000000",
		Amount:  100,
	}
	if err := Validate(tx); err == nil {
		t.Error("expected error for bad card")
	}
}

func TestBatchProcess(t *testing.T) {
	txs := []Transaction{
		{CardNum: "4111111111111111", Amount: 100},
		{CardNum: "0000000000000000", Amount: 200},
		{CardNum: "5500000000000004", Amount: 999999},
	}
	results := BatchProcess(txs)
	if len(results) != 3 {
		t.Fatalf("expected 3 results, got %d", len(results))
	}
	if !results[0].Approved {
		t.Error("first tx should be approved")
	}
	if results[1].Approved {
		t.Error("second tx (bad card) should be rejected")
	}
	if results[2].Approved {
		t.Error("third tx (over limit) should be rejected")
	}
}

func BenchmarkLuhnCheck(b *testing.B) {
	for i := 0; i < b.N; i++ {
		LuhnCheck("4111111111111111")
	}
}

func BenchmarkProcess(b *testing.B) {
	tx := Transaction{CardNum: "4111111111111111", Amount: 100}
	for i := 0; i < b.N; i++ {
		Process(tx)
	}
}

func BenchmarkBatchProcess(b *testing.B) {
	txs := make([]Transaction, 100)
	for i := range txs {
		txs[i] = Transaction{
			CardNum: "4111111111111111",
			Amount:  float64((i + 1) * 100),
		}
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		BatchProcess(txs)
	}
}
