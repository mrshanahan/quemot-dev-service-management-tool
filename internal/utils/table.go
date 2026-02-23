package utils

import (
	"fmt"
	"math"
	"slices"
	"strings"
)

func spaces(n int) string {
	return strings.Repeat(" ", n)
}

func BuildComparisonTable(xheader string, xs []string, yheader string, ys []string) string {
	builder := strings.Builder{}
	writef := func(format string, a ...any) {
		builder.WriteString(fmt.Sprintf(format, a...))
	}
	lenF := func(x string) int { return len(x) }
	var maxNameLen int
	if len(xs) == 0 && len(ys) == 0 {
		maxNameLen = 0
	} else if len(xs) == 0 {
		maxNameLen = slices.Max(Map(ys, lenF))
	} else if len(ys) == 0 {
		maxNameLen = slices.Max(Map(xs, lenF))
	} else {
		maxNameLen = int(math.Max(
			float64(slices.Max(Map(xs, lenF))),
			float64(slices.Max(Map(ys, lenF)))))
	}

	xheaderLen, yheaderLen :=
		int(math.Max(float64(len(xheader)), 3)),
		int(math.Max(float64(len(yheader)), 3))
	xssorted := slices.Clone(xs)
	slices.Sort(xssorted)
	yssorted := slices.Clone(ys)
	slices.Sort(yssorted)

	spaceToXHeaderCenter := xheaderLen / 2
	spaceFromXToYHeaderCenter :=
		(xheaderLen - spaceToXHeaderCenter - 1) + // Remaining space to edge of header col
			1 + // Space between header cols
			yheaderLen/2 // Space from beginning to middle
	spaceFromYHeaderCenter := yheaderLen - yheaderLen/2 - 1

	// - Xs in center of header if possible
	// - 1 space between columns
	//
	// |        HEADER1 HEADER2
	// |        ------- -------
	// |SECRET1    X       X
	// |SECRET2            X
	// |SECRET3    X
	// |FOO                X

	writef("%s %s %s\n", spaces(maxNameLen), xheader, yheader)
	writef("%s %s %s\n", spaces(maxNameLen), strings.Repeat("-", xheaderLen), strings.Repeat("-", yheaderLen))
	var i, j int
	for i < len(xssorted) && j < len(yssorted) {
		if xssorted[i] == yssorted[j] {
			secretName := xssorted[i]
			diffFromMax := maxNameLen - len(secretName)
			writef("%s%s %sX%sX%s\n",
				secretName,
				spaces(diffFromMax),
				spaces(spaceToXHeaderCenter),
				spaces(spaceFromXToYHeaderCenter),
				spaces(spaceFromYHeaderCenter))
			i += 1
			j += 1
		} else if xssorted[i] < yssorted[j] {
			secretName := xssorted[i]
			diffFromMax := maxNameLen - len(secretName)
			writef("%s%s %sX%s %s\n",
				secretName,
				spaces(diffFromMax),
				spaces(spaceToXHeaderCenter),
				spaces(spaceFromXToYHeaderCenter),
				spaces(spaceFromYHeaderCenter))
			i += 1
		} else {
			secretName := yssorted[j]
			diffFromMax := maxNameLen - len(secretName)
			writef("%s%s %s %sX%s\n",
				secretName,
				spaces(diffFromMax),
				spaces(spaceToXHeaderCenter),
				spaces(spaceFromXToYHeaderCenter),
				spaces(spaceFromYHeaderCenter))
			j += 1
		}
	}
	for i < len(xssorted) {
		secretName := xssorted[i]
		diffFromMax := maxNameLen - len(secretName)
		writef("%s%s %sX%s %s\n",
			secretName,
			spaces(diffFromMax),
			spaces(spaceToXHeaderCenter),
			spaces(spaceFromXToYHeaderCenter),
			spaces(spaceFromYHeaderCenter))
		i += 1
	}
	for j < len(yssorted) {
		secretName := yssorted[j]
		diffFromMax := maxNameLen - len(secretName)
		writef("%s%s %s %sX%s\n",
			secretName,
			spaces(diffFromMax),
			spaces(spaceToXHeaderCenter),
			spaces(spaceFromXToYHeaderCenter),
			spaces(spaceFromYHeaderCenter))
		j += 1
	}

	return builder.String()
}
