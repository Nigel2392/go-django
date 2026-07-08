package djester

import (
	"bytes"
	"net/http"
)

type TestResponse struct {
	*http.Response
	buf *bytes.Buffer
}
