// Package pay validates and processes financial transactions.
// Loop Engineering target: maximize transactions/sec throughput.
package pay

import (
	"errors"
	"time"
)

type Transaction struct {
	ID        string
	CardNum   string
	Amount    float64
	Currency  string
	Timestamp time.Time
	Merchant  string
}

type Result struct {
	Transaction Transaction
	Approved    bool
	Reason      string
	Latency     time.Duration
}

// Luhn validates a card number using the Luhn algorithm.
func LuhnCheck(card string) bool {
	if len(card) == 0 {
		return false
	}
	// Reject all-zero cards (0000 passes Luhn but is never valid)
	allZero := true
	for i := 0; i < len(card); i++ {
		if card[i] != '0' {
			allZero = false
			break
		}
	}
	if allZero {
		return false
	}

	var sum int
	alt := false
	for i := len(card) - 1; i >= 0; i-- {
		if card[i] < '0' || card[i] > '9' {
			return false
		}
		d := int(card[i] - '0')
		if alt {
			d *= 2
			if d > 9 {
				d -= 9
			}
		}
		sum += d
		alt = !alt
	}
	return sum%10 == 0
}

// Validate checks a transaction against business rules.
// Returns nil if valid, error describing the failure.
func Validate(tx Transaction) error {
	if tx.Amount <= 0 {
		return errors.New("amount must be positive")
	}
	if tx.Amount > 500000 {
		return errors.New("amount exceeds maximum limit")
	}
	if !LuhnCheck(tx.CardNum) {
		return errors.New("invalid card number")
	}
	if len(tx.CardNum) < 13 || len(tx.CardNum) > 19 {
		return errors.New("invalid card number length")
	}
	return nil
}

// Process validates and approves a transaction.
func Process(tx Transaction) Result {
	start := time.Now()
	err := Validate(tx)
	latency := time.Since(start)
	if err != nil {
		return Result{
			Transaction: tx,
			Approved:    false,
			Reason:      err.Error(),
			Latency:     latency,
		}
	}
	return Result{
		Transaction: tx,
		Approved:    true,
		Reason:      "approved",
		Latency:     latency,
	}
}

// BatchProcess processes multiple transactions and returns results.
func BatchProcess(txs []Transaction) []Result {
	results := make([]Result, len(txs))
	for i, tx := range txs {
		results[i] = Process(tx)
	}
	return results
}
