package bridge

import (
	"errors"
	"testing"
)

func TestOpError_Error(t *testing.T) {
	tests := []struct {
		name string
		err  OpError
		want string
	}{
		{
			name: "without cause",
			err: OpError{
				Class:     FailureTransient,
				Operation: "TestOperation",
			},
			want: "TestOperation: transient",
		},
		{
			name: "with cause",
			err: OpError{
				Class:     FailureUnpaired,
				Operation: "AnotherOperation",
				Cause:     errors.New("connection reset by peer"),
			},
			want: "AnotherOperation: unpaired: connection reset by peer",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.err.Error(); got != tt.want {
				t.Errorf("OpError.Error() = %v, want %v", got, tt.want)
			}
		})
	}
}
