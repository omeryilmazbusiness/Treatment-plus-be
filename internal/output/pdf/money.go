package pdf

import (
	"fmt"
	"strconv"
	"strings"
)

func formatMoney(amount int64, currency, locale string) string {
	neg := amount < 0
	if neg {
		amount = -amount
	}
	whole := amount / 100
	frac := amount % 100
	w := strconv.FormatInt(whole, 10)
	var b strings.Builder
	if locale == "en" {
		for i, c := range w {
			if i > 0 && (len(w)-i)%3 == 0 {
				b.WriteByte(',')
			}
			b.WriteRune(c)
		}
		b.WriteByte('.')
	} else {
		for i, c := range w {
			if i > 0 && (len(w)-i)%3 == 0 {
				b.WriteByte('.')
			}
			b.WriteRune(c)
		}
		b.WriteByte(',')
	}
	fmt.Fprintf(&b, "%02d", frac)
	out := b.String()
	if neg {
		out = "-" + out
	}
	switch currency {
	case "EUR":
		return out + " €"
	case "USD":
		return out + " $"
	default:
		return out + " ₺"
	}
}
