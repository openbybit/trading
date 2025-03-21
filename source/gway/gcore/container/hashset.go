package container

import (
	"fmt"
	"strings"
)

var itemExists = struct{}{}

// HashSet hash set
type HashSet struct {
	Items map[interface{}]struct{}
}

// NewSet new set
func NewSet(values ...interface{}) *HashSet {
	set := &HashSet{Items: make(map[interface{}]struct{})}
	if len(values) > 0 {
		set.Add(values...)
	}
	return set
}

// Add add items to set
func (set *HashSet) Add(items ...interface{}) {
	for _, item := range items {
		if item != nil {
			set.Items[item] = itemExists
		}
	}
}

// Remove remove items from set
func (set *HashSet) Remove(items ...interface{}) {
	for _, item := range items {
		delete(set.Items, item)
	}
}

// Contains check if set contains item
func (set *HashSet) Contains(items ...interface{}) bool {
	for _, item := range items {
		if _, contains := set.Items[item]; !contains {
			return false
		}
	}
	return true
}

// Empty empty set
func (set *HashSet) Empty() bool {
	return set.Size() == 0
}

// Size size of set
func (set *HashSet) Size() int {
	return len(set.Items)
}

// HasAny check if set has any item
func (set *HashSet) HasAny(another *HashSet) bool {
	for item := range another.Items {
		if _, contains := set.Items[item]; contains {
			return true
		}
	}

	return false
}

// Clear clear set
func (set *HashSet) Clear() {
	set.Items = make(map[interface{}]struct{})
}

// Values values of set
func (set *HashSet) Values() []interface{} {
	values := make([]interface{}, set.Size())
	count := 0
	for item := range set.Items {
		values[count] = item
		count++
	}
	return values
}

// String string of set
func (set *HashSet) String() string {
	str := "HashSet\n"
	var items []string
	for k := range set.Items {
		items = append(items, fmt.Sprintf("%v", k))
	}
	str += strings.Join(items, ", ")
	return str
}
