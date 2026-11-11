package creditview

import "strings"

// price is a model's API price in dollars per million tokens. hustleX knows
// Claude's; a plan bills differently, so the page calls it an estimate.
type price struct{ input, output, cacheRead float64 }

// prices go by model ID prefix, the longer first where one starts another.
var prices = []struct {
	prefix string
	price
}{
	{"claude-fable-5-2", price{8, 40, 0.20}},
	{"claude-mythos-5-2", price{8, 40, 0.20}},
	{"claude-fable-5", price{10, 50, 1}},
	{"claude-mythos-5", price{10, 50, 1}},
	{"claude-opus-5-5", price{4, 20, 0.20}},
	{"claude-opus-", price{5, 25, 0.50}},
	{"claude-sonnet-5-5", price{2, 10, 0.10}},
	{"claude-sonnet-5", price{2, 10, 0.20}},
	{"claude-sonnet-", price{3, 15, 0.30}},
	{"claude-haiku-5-5", price{0.10, 0.50, 0.01}},
	{"claude-haiku-", price{1, 5, 0.10}},
}

// cacheWrite is what writing to the cache costs, against input.
const cacheWrite = 1.25

func priceOf(model string) (price, bool) {
	for _, p := range prices {
		if strings.HasPrefix(model, p.prefix) {
			return p.price, true
		}
	}
	return price{}, false
}

// cost is what t would cost of model at API prices; false for a model
// without a known price.
func cost(model string, t usage) (float64, bool) {
	p, ok := priceOf(model)
	if !ok {
		return 0, false
	}
	return (float64(t.input)*p.input + float64(t.output)*p.output + float64(t.cacheRead)*p.cacheRead +
		float64(t.cacheCreate)*p.input*cacheWrite) / 1e6, true
}

// cacheSaving is what reading t's cache saved against reading it as new
// input, at API prices.
func cacheSaving(model string, t usage) float64 {
	p, ok := priceOf(model)
	if !ok {
		return 0
	}
	return float64(t.cacheRead) * (p.input - p.cacheRead) / 1e6
}
