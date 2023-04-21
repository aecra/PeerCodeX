package tools

import (
	"fmt"
	"math"
)

func FormatByteSize(bytes int64) string {
	units := []string{"B", "KB", "MB", "GB", "TB", "PB", "EB"}
	base := int64(1024)

	if bytes == 0 {
		return "0 B"
	}

	exponent := int64(math.Log(float64(bytes)) / math.Log(float64(base)))
	unit := units[exponent]

	result := float64(bytes) / math.Pow(float64(base), float64(exponent))
	formatted := fmt.Sprintf("%.2f %s", result, unit)

	return formatted
}
