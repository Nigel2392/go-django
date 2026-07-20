//go:build test
// +build test

package djester

import (
	"bytes"
	"encoding/json"
	"net/http"

	"github.com/PuerkitoBio/goquery"
)

type TestResponse struct {
	*http.Response
	buf  []byte
	html *goquery.Document
	t    *Tester
}

func (t *TestResponse) setupBuffer() (err error) {
	if t.buf != nil {
		return nil
	}

	buf := new(bytes.Buffer)
	_, err = buf.ReadFrom(t.Response.Body)
	if err != nil {
		return err
	}

	t.buf = buf.Bytes()
	return t.Response.Body.Close()
}

func (r *TestResponse) Assert(t BaseTB, verbose bool) ResponseAssertion {
	return &responseAssertion{
		assertion: assertion[BaseTB]{
			test:    t,
			verbose: verbose,
		},
		response: r,
	}
}

func (t *TestResponse) JSON(scanTo interface{}) (err error) {
	if err = t.setupBuffer(); err != nil {
		return err
	}
	var enc = json.NewDecoder(bytes.NewReader(t.buf))
	err = enc.Decode(scanTo)
	return err
}

func (t *TestResponse) HTML() (string, error) {
	if err := t.setupBuffer(); err != nil {
		return "", err
	}

	return string(t.buf), nil
}

func (t *TestResponse) DOM() (doc *goquery.Document, err error) {
	if t.html != nil {
		return t.html, nil
	}

	if err := t.setupBuffer(); err != nil {
		return nil, err
	}

	t.html, err = goquery.NewDocumentFromReader(bytes.NewReader(t.buf))
	return t.html, err
}
