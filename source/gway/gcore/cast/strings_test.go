package cast

import (
	"bytes"
	"reflect"
	"testing"
)

func TestBytesToString(t *testing.T) {
	a := []byte{'h', 'e', 'l', 'l', 'o'}
	b := UnsafeBytesToString(a)
	if b != "hello" {
		t.Error("UnsafeBytesToString fail")
	}
}

func TestStringToBytes(t *testing.T) {
	a := "hello"
	b := UnsafeStringToBytes(a)
	if !bytes.Equal(b, []byte("hello")) {
		t.Error("invalid")
	} else {
		t.Logf("%s", b)
	}
}

func TestInt642Bytes(t *testing.T) {
	type args struct {
		n int64
	}
	tests := []struct {
		name string
		args args
		want []byte
	}{
		// TODO: Add test cases.
		{
			name: "0",
			args: args{
				n: 0,
			},
			want: []byte{0, 0, 0, 0, 0, 0, 0, 0},
		},
		{
			name: "1",
			args: args{
				n: 1,
			},
			want: []byte{0, 0, 0, 0, 0, 0, 0, 1},
		},
		{
			name: "114561287642",
			args: args{
				n: 114561287642,
			},
			want: []byte{0, 0, 0, 26, 172, 98, 133, 218},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Int64ToBytes(tt.args.n); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Int642Bytes() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestBytes2Int64(t *testing.T) {
	type args struct {
		b []byte
	}
	tests := []struct {
		name string
		args args
		want int64
	}{
		// TODO: Add test cases.
		{
			name: "",
			args: args{
				b: []byte{0, 0, 0, 26, 172, 98, 133, 218},
			},
			want: 114561287642,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := BytesToInt64(tt.args.b); got != tt.want {
				t.Errorf("Bytes2Int64() = %v, want %v", got, tt.want)
			}
		})
	}
}
