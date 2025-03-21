package conhash

import (
	"hash/crc32"
	"reflect"
	"testing"
)

func TestAdd(t *testing.T) {
	cycle1 := New()
	cycle2 := New()

	cycle1.Add("A")
	cycle1.Add("B")
	cycle1.Add("C")
	cycle1.Add("D")
	cycle1.Add("E")
	cycle1.Add("F")

	cycle2.Set([]string{"A", "B", "C", "D", "E", "F"})

	n1, err1 := cycle1.Get("target")
	n2, err2 := cycle2.Get("target")
	if err1 != nil || err2 != nil {
		t.Errorf("should be nil, err1=%v, err2=%v", err1, err2)
	}

	if !reflect.DeepEqual(n1, n2) {
		t.Errorf("should be equal, n1=%v, n2=%v", n1, n2)
	}
}

func TestRemove(t *testing.T) {
	cycle1 := New()
	cycle2 := New()

	cycle1.Add("A")
	cycle1.Add("B")
	cycle1.Add("C")
	cycle1.Add("D")
	cycle1.Add("E")
	cycle1.Add("F")
	cycle1.Remove("A")
	cycle1.Remove("B")
	cycle1.Remove("C")

	cycle2.Set([]string{"A", "B", "C", "D", "E", "F"})
	cycle2.Set([]string{"D", "E", "F"})

	n1, err1 := cycle1.Get("target")
	n2, err2 := cycle2.Get("target")
	if err1 != nil || err2 != nil {
		t.Errorf("should be nil, err1=%v, err2=%v", err1, err2)
	}

	if !reflect.DeepEqual(n1, n2) {
		t.Errorf("should be equal, n1=%v, n2=%v", n1, n2)
	}
}

func TestAddRemove(t *testing.T) {
	cycle1 := New()
	cycle2 := New()

	cycle1.Add("A")
	cycle1.Add("B")
	cycle1.Add("C")
	cycle1.Add("D")
	cycle1.Add("E")
	cycle1.Add("F")
	cycle1.Remove("A")
	cycle1.Remove("B")
	cycle1.Remove("C")
	cycle1.Add("a")
	cycle1.Add("b")
	cycle1.Add("c")

	cycle2.Set([]string{"A", "B", "C", "D", "E", "F"})
	cycle2.Set([]string{"D", "E", "F", "a", "b", "c"})

	n1, err1 := cycle1.Get("target")
	n2, err2 := cycle2.Get("target")
	if err1 != nil || err2 != nil {
		t.Errorf("should be nil, err1=%v, err2=%v", err1, err2)
	}

	if !reflect.DeepEqual(n1, n2) {
		t.Errorf("should be equal, n1=%v, n2=%v", n1, n2)
	}
}

var key = "1234567890"

func BenchmarkConvert(b *testing.B) {
	for i := 0; i < b.N; i++ {
		c := []byte(key)
		_ = crc32.ChecksumIEEE(c)
	}
}

func BenchmarkCopy(b *testing.B) {
	for i := 0; i < b.N; i++ {
		var s [64]byte
		copy(s[:], key)
		_ = crc32.ChecksumIEEE(s[:len(key)])
	}
}
