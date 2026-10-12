package cmd

import (
	"errors"
	"fmt"
	"testing"
)

func TestOperatorInstruction(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want string
	}{
		{
			name: "nil error",
			err:  nil,
			want: "",
		},
		{
			name: "unrelated error",
			err:  errors.New("some unrelated error"),
			want: "",
		},
		{
			name: "ambiguous error",
			err:  ambiguousSendError("foo", errors.New("underlying cause")),
			want: "Do not send with a new key. To replay-check this exact invocation, repeat it with the same idempotency key: foo",
		},
		{
			name: "wrapped ambiguous error",
			err:  fmt.Errorf("wrapped: %w", ambiguousSendError("bar", errors.New("underlying cause"))),
			want: "Do not send with a new key. To replay-check this exact invocation, repeat it with the same idempotency key: bar",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := OperatorInstruction(tt.err)
			if got != tt.want {
				t.Errorf("OperatorInstruction() = %q, want %q", got, tt.want)
			}
		})
	}
}
