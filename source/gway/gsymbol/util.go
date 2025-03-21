package gsymbol

import "strings"

func calcScale(x string) int {
	idx := strings.IndexByte(x, '.')
	if idx == -1 {
		return 0
	}

	for i := len(x) - 1; i > idx; i-- {
		if x[i] != '0' {
			return i - idx
		}
	}

	return 0
}
