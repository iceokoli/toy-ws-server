package stream

import (
	"math"
	"testing"
)

const float64EqualityThreshold = 1e-9

func almostEqual(a, b float64) bool {
	return math.Abs(a-b) <= float64EqualityThreshold
}

func TestGetPrice(t *testing.T) {
	t.Parallel()

	gen := NewPriceGenerator()
	symbol := "BTC"
	prices := [5]float64{}
	for i := 0; i < 5; i++ {
		if i > 0 {
			if !almostEqual(prices[i-1], gen.config[symbol].price) {
				t.Errorf("Failed to update config price")
			}
		}
		newPrice, err := gen.GetPrice(symbol)
		if err != nil {
			t.Errorf("Failed to get price for %s, err=%v", symbol, err)
		}
		prices[i] = newPrice
	}
}