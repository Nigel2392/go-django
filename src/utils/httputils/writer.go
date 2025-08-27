package httputils

import (
	"io"
	"net/http"
)

type WriterCopyFlag int

const (
	FlagCopyHeader WriterCopyFlag = iota << 1
	FlagCopyBody
	FlagCopyStatus

	FlagCopyAll = FlagCopyHeader | FlagCopyBody | FlagCopyStatus
)

var _ http.ResponseWriter = (*FakeWriter[io.ReadWriter])(nil)

type FakeWriter[DST io.ReadWriter] struct {
	WriteTo    DST
	Headers    http.Header
	StatusCode int
}

func NewFakeWriter[DST io.ReadWriter](dst DST) *FakeWriter[DST] {
	return &FakeWriter[DST]{
		WriteTo:    dst,
		Headers:    make(http.Header),
		StatusCode: 0,
	}
}

func (w *FakeWriter[DST]) Header() http.Header {
	return w.Headers
}

func (w *FakeWriter[DST]) WriteHeader(statusCode int) {
	w.StatusCode = statusCode
}

func (w *FakeWriter[DST]) Write(b []byte) (int, error) {
	return w.WriteTo.Write(b)
}

func (w *FakeWriter[DST]) CopyTo(wr http.ResponseWriter, flags ...WriterCopyFlag) (int64, error) {
	var flag WriterCopyFlag
	if len(flags) == 0 {
		flag = FlagCopyAll
	} else {
		for _, f := range flags {
			flag |= f
		}
	}

	if flag&FlagCopyStatus != 0 {
		wr.WriteHeader(w.StatusCode)
	}

	if flag&FlagCopyHeader != 0 {
		for key, values := range w.Headers {
			for _, value := range values {
				wr.Header().Add(key, value)
			}
		}
	}

	if flag&FlagCopyBody != 0 {
		if _, err := io.Copy(wr, w.WriteTo); err != nil {
			return 0, err
		}
	}

	return 0, nil
}

func (w *FakeWriter[DST]) Unwrap() DST {
	return w.WriteTo
}
