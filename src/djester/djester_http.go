//go:build test
// +build test

package djester

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"mime/multipart"
	"net/http"
	"net/url"
	url_URL "net/url"
	"os"
	"reflect"
)

func (d *Tester) makeRequest(method string, url string, headers http.Header, params url.Values, body io.Reader, extra func(r *http.Request) error) (*http.Response, error) {
	var baseURL, err = url_URL.Parse(d.testServer.URL)
	if err != nil {
		return nil, err
	}
	baseURL.Path = url
	baseURL.RawQuery = params.Encode()

	req, err := http.NewRequest(
		method, baseURL.String(), body,
	)
	if err != nil {
		return nil, err
	}

	for key, values := range headers {
		for _, value := range values {
			req.Header.Add(key, value)
		}
	}

	if extra != nil {
		if err := extra(req); err != nil {
			return nil, err
		}
	}

	return d.testClient.Do(req)
}

func (d *Tester) makeJsonRequest(method string, url string, headers http.Header, params url.Values, body any, scanTo any) (*http.Response, error) {
	var (
		b   *bytes.Buffer = nil
		err error
	)

	if body != nil {
		b = new(bytes.Buffer)
		if r, ok := body.(io.Reader); ok {
			_, err = b.ReadFrom(r)
		} else {
			var enc = json.NewEncoder(b)
			enc.SetIndent("", "  ")
			err = enc.Encode(body)
		}
	}

	if err != nil {
		return nil, err
	}

	var resp *http.Response
	resp, err = d.makeRequest(method, url, headers, params, b, func(r *http.Request) error {
		r.Header.Set("Content-Type", "application/json")
		return nil
	})
	if err != nil {
		return nil, err
	}

	if scanTo != nil {
		if resp.Body == nil {
			return nil, ErrNoResponseBody
		}

		if err := json.NewDecoder(resp.Body).Decode(scanTo); err != nil {
			return nil, err
		}
	}
	return resp, nil
}

func (d *Tester) makeFormRequest(url string, headers http.Header, params url.Values, form map[string]interface{}) (*http.Response, error) {
	var err error
	var b = new(bytes.Buffer)
	var mw = multipart.NewWriter(b)
	for key, value := range form {
		var fw io.Writer
		if x, ok := value.(io.Closer); ok {
			defer x.Close()
		}

		var w io.Reader
		switch t := value.(type) {
		case *os.File:
			fw, err = mw.CreateFormFile(key, t.Name())
			w = t
		case fs.File:
			var stat, err = t.Stat()
			if err != nil {
				mw.Close()
				return nil, err
			}
			fw, err = mw.CreateFormFile(key, stat.Name())
			w = t
		case string:
			fw, err = mw.CreateFormField(key)
			w = bytes.NewBufferString(t)
		case []byte:
			fw, err = mw.CreateFormField(key)
			w = bytes.NewBuffer(t)
		case io.Reader:
			fw, err = mw.CreateFormField(key)
			w = t
		default:
			var rV = reflect.ValueOf(value)
			switch rV.Kind() {
			case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
				fw, err = mw.CreateFormField(key)
				w = bytes.NewBufferString(fmt.Sprintf("%d", rV.Int()))
			case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
				fw, err = mw.CreateFormField(key)
				w = bytes.NewBufferString(fmt.Sprintf("%d", rV.Uint()))
			case reflect.Float32, reflect.Float64:
				fw, err = mw.CreateFormField(key)
				w = bytes.NewBufferString(fmt.Sprintf("%f", rV.Float()))
			case reflect.String:
				fw, err = mw.CreateFormField(key)
				w = bytes.NewBufferString(rV.String())
			}
		}
		if err != nil {
			mw.Close()
			return nil, err
		}

		if _, err = io.Copy(fw, w); err != nil {
			mw.Close()
			return nil, err
		}
	}

	if err = mw.Close(); err != nil {
		return nil, err
	}

	return d.makeRequest(http.MethodPost, url, headers, params, b, func(r *http.Request) error {
		r.Header.Set("Content-Type", mw.FormDataContentType())
		return nil
	})
}

func (d *Tester) Get(url string, headers http.Header, params url.Values) (*http.Response, error) {
	return d.makeRequest(http.MethodGet, url, headers, params, nil, nil)
}

func (d *Tester) GetJson(url string, headers http.Header, params url.Values, scanTo any) (*http.Response, error) {
	return d.makeJsonRequest(http.MethodGet, url, headers, params, nil, scanTo)
}

func (d *Tester) Post(url string, headers http.Header, params url.Values, body io.Reader) (*http.Response, error) {
	return d.makeRequest(http.MethodPost, url, headers, params, body, nil)
}

func (d *Tester) PostForm(url string, headers http.Header, params url.Values, form map[string]interface{}) (*http.Response, error) {
	return d.makeFormRequest(url, headers, params, form)
}

func (d *Tester) PostFile(url string, headers http.Header, params url.Values, fileName string, file io.Reader) (*http.Response, error) {
	return d.makeFormRequest(url, headers, params, map[string]interface{}{fileName: file})
}

func (d *Tester) PostJson(url string, headers http.Header, params url.Values, body any, scanTo any) (*http.Response, error) {
	return d.makeJsonRequest(http.MethodPost, url, headers, params, body, scanTo)
}
