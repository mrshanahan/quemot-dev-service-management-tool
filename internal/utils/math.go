package utils

import "math"

func IntMax(xs ...int) int {
	if len(xs) == 0 {
		panic("IntMax requires at least one argument")
	}
	max := math.MinInt
	for _, x := range xs {
		if x > max {
			max = x
		}
	}
	return max
}
