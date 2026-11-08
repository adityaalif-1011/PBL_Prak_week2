package helper

import (
	"strconv"
	"strings"

	"github.com/gofiber/fiber/v2"
)

// Format representasi yang didukung endpoint list.
const (
	FormatJSON = "json"
	FormatCSV  = "csv"
)

// Negotiate memilih representasi response dari header Accept
// (dan query ?format=csv|json). Default selalu json.
func Negotiate(c *fiber.Ctx) string {
	if format := strings.ToLower(strings.TrimSpace(c.Query("format"))); format != "" {
		if format == FormatCSV {
			return FormatCSV
		}
		if format == FormatJSON {
			return FormatJSON
		}
	}

	accept := c.Get(fiber.HeaderAccept)
	if accept == "" {
		return FormatJSON
	}

	bestQ := 0.0
	best := FormatJSON
	for _, part := range strings.Split(accept, ",") {
		media := strings.ToLower(strings.TrimSpace(part))
		if media == "" {
			continue
		}

		name, q := parseMediaRange(media)
		switch name {
		case "text/csv", "application/csv":
			if q > bestQ {
				bestQ, best = q, FormatCSV
			}
		case "application/json", "*/*", "application/*":
			if q > bestQ {
				bestQ, best = q, FormatJSON
			}
		}
	}

	return best
}

// parseMediaRange memisahkan "text/csv;q=0.9" menjadi name dan q.
func parseMediaRange(media string) (string, float64) {
	name := media
	q := 1.0

	if idx := strings.Index(media, ";"); idx >= 0 {
		name = strings.TrimSpace(media[:idx])
		for _, param := range strings.Split(media[idx+1:], ";") {
			kv := strings.SplitN(strings.TrimSpace(param), "=", 2)
			if len(kv) == 2 && strings.EqualFold(kv[0], "q") {
				if parsed, err := strconv.ParseFloat(strings.Trim(kv[1], `"`), 64); err == nil {
					q = parsed
				}
			}
		}
	}
	return name, q
}
