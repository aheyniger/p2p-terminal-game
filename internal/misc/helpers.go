package misc

import (
	"fmt"
)

// Helper func to convert string → int
func MustAtoi(s string) int {
	var i int
	fmt.Sscanf(s, "%d", &i)
	return i
}

func Atoi32(s string) int32 {
	return int32(MustAtoi(s))
}
