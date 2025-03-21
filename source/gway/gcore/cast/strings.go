package cast

import (
	"bytes"
	"encoding/binary"
	"strconv"
	"unsafe"
)

// StringToBool string convert to bool
func StringToBool(s string) bool {
	b, _ := strconv.ParseBool(s)
	return b
}

// Atoi string to int
func Atoi(s string) int {
	i, _ := strconv.Atoi(s)
	return i
}

// AtoInt64 string to int64
func AtoInt64(s string) int64 {
	i, _ := strconv.ParseInt(s, 10, 64)
	return i
}

// Itoa int to string
func Itoa(i int) string {
	return strconv.Itoa(i)
}

// Int64toa int64 to string
func Int64toa(i int64) string {
	return strconv.FormatInt(i, 10)
}

// Int64ToBytes covert int to bytes.
func Int64ToBytes(n int64) []byte {
	buf := bytes.NewBuffer([]byte{})
	// byteOrder big-endian
	if err := binary.Write(buf, binary.BigEndian, n); err != nil {
		return nil
	}
	return buf.Bytes()
}

// BytesToInt64 covert bytes to int .
func BytesToInt64(b []byte) int64 {
	bytesBuffer := bytes.NewBuffer(b)
	var x int64
	// byteOrder big-endian
	if err := binary.Read(bytesBuffer, binary.BigEndian, &x); err != nil {
		return 0
	}
	return x
}

// UnsafeBytesToString unsafe convert string to byte
func UnsafeStringToBytes(s string) (b []byte) {
	// https://www.sobyte.net/post/2022-09/string-byte-convertion/
	x := (*[2]uintptr)(unsafe.Pointer(&s))
	h := [3]uintptr{x[0], x[1], x[1]}
	return *(*[]byte)(unsafe.Pointer(&h))
	// sh := (*reflect.StringHeader)(unsafe.Pointer(&s))
	// bh := (*reflect.SliceHeader)(unsafe.Pointer(&b))

	// bh.Data, bh.Len, bh.Cap = sh.Data, sh.Len, sh.Len
	// return b
}

// UnsafeBytesToString unsafe convert bytes to string
func UnsafeBytesToString(b []byte) string {
	return *(*string)(unsafe.Pointer(&b))
}
