package util

import "io"

// CopyBuffer is a copy-on-read buffer.
type CopyBuffer struct {
	// Cursor tracks progress through the buffer between read calls.
	Cursor int
	// Data is the underlying data to copy out.
	Data []byte
}

// NewCopyBuffer allocates a copy buffer seeded by the provided byte slice.
func NewCopyBuffer(buffer []byte) *CopyBuffer {
	return &CopyBuffer{
		Data: buffer,
	}
}

// Read until EOF. Subsequent reads will reset to the start of the buffer.
func (b *CopyBuffer) Read(p []byte) (int, error) {
	if b.Data == nil {
		return 0, io.EOF
	}
	// Cap the end by the length of the data to avoid out of bounds.
	end := len(p) + b.Cursor
	if end > len(b.Data) {
		end = len(b.Data)
	}
	// Copy the data into the out buffer, advancing the cursor by the amount
	// copied.
	n := copy(p, b.Data[b.Cursor:end])
	b.Cursor += n
	// Reset the cursor once the buffer has been fully traversed.
	if b.Cursor >= len(b.Data) {
		b.Cursor = 0
		return n, io.EOF
	}
	return n, nil
}
