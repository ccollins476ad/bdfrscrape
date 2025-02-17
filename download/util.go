package download

import (
	"context"
	"io"
)

// ContextRead calls r.Read() with respect to the given context. It orphans an
// active read in a separate goroutine if the context finishes early.
func ContextRead(ctx context.Context, r io.Reader, p []byte) (int, error) {
	type Result struct {
		n   int
		err error
	}

	resultChan := make(chan Result, 1)

	go func() {
		defer close(resultChan)
		n, err := r.Read(p)
		resultChan <- Result{n, err}
	}()

	select {
	case <-ctx.Done():
		return 0, ctx.Err()
	case result := <-resultChan:
		return result.n, result.err
	}
}

type ContextReader struct {
	ctx context.Context
	r   io.Reader
}

func NewContextReader(ctx context.Context, r io.Reader) *ContextReader {
	return &ContextReader{
		ctx: ctx,
		r:   r,
	}
}

// Read implements io.Reader#Read(), respecting the ContextReader's embedded
// context.
func (cr *ContextReader) Read(p []byte) (int, error) {
	return ContextRead(cr.ctx, cr.r, p)
}
