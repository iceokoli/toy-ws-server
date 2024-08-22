package stream

import (
	"fmt"
	"math"
	"math/rand/v2"
)

var SYMBOLS []string = []string{"BTC", "ETH", "SOL"}

type PriceGeneratorConfig struct {
	price     float64
	drift     float64
	diffusion float64
}

type PriceGeneratorConfigs map[string]PriceGeneratorConfig

type PriceGenerator struct {
	symbols map[string]bool
	config  PriceGeneratorConfigs
}

func NewPriceGenerator() *PriceGenerator {
	s := map[string]bool{}
	for _, symbol := range SYMBOLS {
		s[symbol] = true
	}

	cfg := make(PriceGeneratorConfigs)
	cfg["BTC"] = PriceGeneratorConfig{price: 60_000, drift: 0.02, diffusion: 0.04}
	cfg["ETH"] = PriceGeneratorConfig{price: 3_000, drift: 0.02, diffusion: 0.04}
	cfg["SOL"] = PriceGeneratorConfig{price: 142, drift: 0.02, diffusion: 0.04}

	return &PriceGenerator{symbols: s, config: cfg}
}

func (r *PriceGenerator) GetPrice(symbol string) (float64, error) {
	cfg, ok := r.config[symbol]
	if !ok {
		return 0, fmt.Errorf("%v not supported", symbol)
	}

	dt := 1.0
	powerA := (cfg.drift - 0.5*math.Pow(cfg.diffusion, 2)) * dt
	powerB := cfg.diffusion * math.Sqrt(dt) * rand.NormFloat64()
	currentPrice := cfg.price
	newPrice := currentPrice * math.Exp(powerA+powerB)

	cfg.price = newPrice
	r.config[symbol] = cfg
	return newPrice, nil
}
