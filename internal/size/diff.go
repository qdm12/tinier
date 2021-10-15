package size

import (
	"fmt"
)

func DiffString(outputSize, inputSize int64) (s string) {
	if inputSize == 0 {
		return "0B ➡️  0B (+0%)"
	}
	ratio := float64(outputSize) / float64(inputSize)
	diff := 100 * (ratio - 1) //nolint:gomnd
	sign := "+"
	if diff < 0 {
		diff = -diff
		sign = "-"
	}
	diffString := fmt.Sprintf("%s%.1f%%", sign, diff)

	outputHumanSize := BytesToHuman(outputSize)
	inputHumanSize := BytesToHuman(inputSize)

	return fmt.Sprintf("%s ➡️  %s (%s)",
		inputHumanSize, outputHumanSize, diffString)
}
