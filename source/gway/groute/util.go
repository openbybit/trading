package groute

import (
	"sort"
	"strings"
)

// OrderedList 要求是有序列表
type OrderedList []string

func (l *OrderedList) Sort() {
	sort.Strings(*l)
}

// Contains 二分查找判断是否存在
func (l OrderedList) Contains(x string) bool {
	idx := sort.SearchStrings(l, x)
	if idx < len(l) && l[idx] == x {
		return true
	}

	return false
}

// ContainsAny 判断是否包含另一个数组中任意一个
func (l OrderedList) ContainsAny(o []string) bool {
	for _, x := range o {
		if l.Contains(x) {
			return true
		}
	}

	return false
}

// Equal 比较两个有序列表内容是否完全一样
func (l OrderedList) Equal(o OrderedList) bool {
	if len(l) != len(o) {
		return false
	}

	for idx, x := range l {
		if x != o[idx] {
			return false
		}
	}

	return true
}

func (l OrderedList) String() string {
	return strings.Join(l, ",")
}

func trimPath(path string) string {
	if path == "/" {
		return path
	}
	return strings.TrimSuffix(path, "/")
}
