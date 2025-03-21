package berror

import (
	"errors"
	"reflect"
	"testing"
)

func TestNewError(t *testing.T) {
	type args struct {
		code    int64
		message []string
	}
	tests := []struct {
		name string
		args args
		want BizErr
	}{
		// TODO: Add test cases.
		{
			name: "default",
			args: args{
				code:    0,
				message: []string{},
			},
			want: BizErr{
				baseErr: baseErr{
					Code:    0,
					Message: "",
				},
			},
		},
		{
			name: "code-100",
			args: args{
				code:    100,
				message: []string{"a", "b"},
			},
			want: BizErr{
				baseErr: baseErr{
					Code:    100,
					Message: "a, b",
				},
			},
		},
		{
			name: "code-1",
			args: args{
				code:    -1,
				message: []string{"a"},
			},
			want: BizErr{
				baseErr: baseErr{
					Code:    -1,
					Message: "a",
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := NewBizErr(tt.args.code, tt.args.message...); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewBizErr() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestBError_Error(t *testing.T) {
	var err error
	err = BizErr{
		baseErr: baseErr{
			Code:    123,
			Message: "rrrr",
		},
	}

	be := &BizErr{}
	b := errors.As(err, be)
	t.Log(b, be.Code)

	be = &BizErr{}
	b = errors.As(err, be)
	t.Log(b, be.Code)

	be = &BizErr{}
	err = WithMessage(err, "abc")
	b = errors.As(err, be)
	t.Log(b, be.Code)
}
