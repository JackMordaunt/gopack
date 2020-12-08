package util

import (
	"io"
	"testing"
)

// TestCopyBuffer ensures that copy buffers are not consumed when read and are
// properly reset when read fully.
func TestCopyBuffer(t *testing.T) {
	tests := []struct {
		Input    []byte
		ReadSize int
		ReReads  int
	}{
		{
			Input:    []byte("123456"),
			ReadSize: 2,
			ReReads:  3,
		},
		{
			Input:    []byte("123456"),
			ReadSize: 6,
			ReReads:  3,
		},
		{
			Input:    []byte("123456"),
			ReadSize: 5,
			ReReads:  1,
		},
	}
	for _, tt := range tests {
		var (
			output = make([]byte, len(tt.Input))
			cb     = NewCopyBuffer(tt.Input)
		)
		for ii := 0; ii < tt.ReReads; ii++ {
			_, err := io.ReadFull(cb, output)
			if err != nil {
				t.Fatalf("unexpected error while reading from buffer: %v\n", err)
			}
			// Assert data not consumed by read.
			if got, want := string(output), string(cb.Data); got != want {
				t.Fatalf("data read and data in buffer not equal got=%s, want=%s\n", got, want)
			}
			// Assert cursor is reset after full read.
			if cb.Cursor != 0 {
				t.Fatalf("cursor did not reset after full read got=%d, want=%d\n", cb.Cursor, 0)
			}
		}
	}
}
