package size

import "fmt"

func BytesToHuman(bytes int64) (s string) {
	const (
		KB = 1024
		MB = KB * 1024
		GB = MB * 1024
	)
	switch {
	case bytes < 5*KB:
		return fmt.Sprintf("%dB", bytes)
	case bytes < 5*MB:
		return fmt.Sprintf("%dKB", bytes/KB)
	case bytes < 5*GB:
		return fmt.Sprintf("%dMB", bytes/MB)
	default:
		return fmt.Sprintf("%dGB", bytes/GB)
	}
}
