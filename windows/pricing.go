package main

import "strings"

type priceRate struct {
	in, cacheW, cacheR, out float64
}

func estimateUSD(name string, in, out, cacheR, cacheW float64) (float64, bool) {
	r, ok := bundledRate(name)
	if !ok {
		return 0, false
	}
	usd := in/1e6*r.in + cacheW/1e6*r.cacheW + cacheR/1e6*r.cacheR + out/1e6*r.out
	return usd, true
}

func bundledRate(name string) (priceRate, bool) {
	key := strings.ToLower(name)
	if strings.HasPrefix(key, "sand-") || key == "default" {
		return priceRate{}, false
	}
	has := func(s string) bool { return strings.Contains(key, s) }
	switch {
	case has("composer-2.5") && has("fast"):
		return priceRate{3, 0, 0.5, 15}, true
	case has("composer-2.5"):
		return priceRate{0.5, 0, 0.2, 2.5}, true
	case has("grok-4.6") && has("fast"):
		return priceRate{4, 0, 1, 12}, true
	case has("grok-4.6"):
		return priceRate{2, 0, 0.5, 6}, true
	case has("grok-4.5") && has("fast"):
		return priceRate{4, 0, 1, 12}, true
	case has("grok-4.5"):
		return priceRate{2, 0, 0.5, 6}, true
	case has("opus-4-8") || has("opus-4.8") || has("opus-5"):
		return priceRate{5, 6.25, 0.5, 25}, true
	case has("4.5-sonnet") || has("sonnet-4.5") || has("4-5-sonnet") || has("sonnet-4-5") || has("sonnet-5"):
		return priceRate{3, 3.75, 0.3, 15}, true
	case has("gemini-3.1-pro"):
		return priceRate{2, 0, 0.2, 12}, true
	case has("gpt-5") || has("gpt-4.1") || has("gpt-4o"):
		return priceRate{2.5, 0, 0.25, 10}, true
	}
	return priceRate{}, false
}

func formatEstimate(n float64, ok bool) string {
	if !ok {
		return "无标价"
	}
	return "预估 " + formatMoney(n)
}

func formatOver(used, limit float64) string {
	if limit <= 0 {
		return formatMoney(used)
	}
	if used > limit {
		return formatMoney(used) + " / " + formatMoney(limit) + "  超出 " + formatMoney(used-limit)
	}
	return formatMoney(used) + " / " + formatMoney(limit)
}
