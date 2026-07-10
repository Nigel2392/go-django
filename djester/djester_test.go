//go:build test
// +build test

package djester_test

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"testing"

	"github.com/Nigel2392/go-django/djester"
	django "github.com/Nigel2392/go-django/src"
	"github.com/Nigel2392/go-django/src/apps"
	"github.com/Nigel2392/mux"
)

func newApp() *apps.AppConfig {
	app := apps.NewAppConfig("djester_selftest")
	app.Routing = func(m mux.Multiplexer) {
		m.Handle("GET", "/ping", mux.NewHandler(func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte("pong"))
		}))

		// NEW ROUTE: Serve HTML for our DOM tests
		m.Handle("GET", "/html", mux.NewHandler(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			htmlBody := `
			<!DOCTYPE html>
			<html>
			<head><title>Test Dashboard</title></head>
			<body>
				<nav class="main-menu">
					<ul>
						<li>Home</li>
						<li>About</li>
						<li>Contact</li>
					</ul>
				</nav>
				<h1 class="page-title" data-custom="title-data">Dashboard</h1>
				<form id="login-form" method="POST" action="/login">
					<input type="text" name="user" required>
					<button type="submit">Login</button>
				</form>
			</body>
			</html>
			`
			w.Write([]byte(htmlBody))
		}))

		m.Handle("POST", "/echo", mux.NewHandler(func(w http.ResponseWriter, r *http.Request) {
			b, _ := io.ReadAll(r.Body)
			defer r.Body.Close()
			w.Write(b)
		}))

		m.Handle("POST", "/json", mux.NewHandler(func(w http.ResponseWriter, r *http.Request) {
			if !strings.HasPrefix(r.Header.Get("Content-Type"), "application/json") {
				http.Error(w, "Bad Request", http.StatusBadRequest)
				return
			}
			b, _ := io.ReadAll(r.Body)
			defer r.Body.Close()
			w.Header().Set("Content-Type", "application/json")
			w.Write(b)
		}))

		m.Handle("POST", "/form", mux.NewHandler(func(w http.ResponseWriter, r *http.Request) {
			if !strings.HasPrefix(r.Header.Get("Content-Type"), "multipart/form-data") {
				http.Error(w, "Bad Request", http.StatusBadRequest)
				return
			}
			r.ParseMultipartForm(1024)
			val := r.FormValue("foo")
			w.Write([]byte(val))
		}))

		m.Handle("POST", "/upload", mux.NewHandler(func(w http.ResponseWriter, r *http.Request) {
			if !strings.HasPrefix(r.Header.Get("Content-Type"), "multipart/form-data") {
				http.Error(w, "Bad Request", http.StatusBadRequest)
				return
			}
			r.ParseMultipartForm(1024)
			file, _, err := r.FormFile("file.txt")
			if err != nil {
				w.WriteHeader(400)
				w.Write([]byte("no file"))
				return
			}
			defer file.Close()
			data, _ := io.ReadAll(file)
			w.Write(data)
		}))
	}
	return app
}

func TestDjester(t *testing.T) {
	d := &djester.Tester{
		Settings: map[string]any{
			django.APPVAR_ALLOWED_HOSTS:  []string{"*"},
			django.APPVAR_DISABLE_NOSURF: true,
			django.APPVAR_DEBUG:          true,
		},
		Flags: []django.AppFlag{
			django.FlagSkipCmds,
			django.FlagSkipChecks,
		},
		Apps: []djester.AppInitFuncOrAppConfig{
			newApp,
		},
		Tests: []djester.Test{
			&djester.BasicTest{
				Label: "Assert suite works",
				Function: func(d *djester.Tester, t *testing.T) {
					assert := d.Assert(t, true)
					assert.AssertEqual(1, 1)
					assert.AssertNotEqual("x", "y")
					assert.AssertNil(nil)
					assert.AssertNotNil(42)
					assert.AssertContains([]string{"a", "b"}, "a")
					assert.AssertNotContains([]int{1, 2}, 3)
				},
			},
			&djester.BasicTest{
				Label: "GET request works",
				Function: func(d *djester.Tester, t *testing.T) {
					assert := d.Assert(t, true)
					resp, err := d.Get("/ping", nil, url.Values{})
					assert.Assert(err == nil, "GET failed: %v", err)
					defer resp.Body.Close()
					b, _ := io.ReadAll(resp.Body)
					assert.AssertEqual("pong", string(b))
				},
			},
			// NEW TEST: HTML Assertions
			&djester.BasicTest{
				Label: "HTML assertions work",
				Function: func(d *djester.Tester, t *testing.T) {
					// 1. Fetch the HTML
					resp, err := d.Get("/html", nil, url.Values{})
					d.Assert(t, true).Assert(err == nil, "GET /html failed: %v", err)
					defer resp.Body.Close()

					// 2. Get the ResponseAssertion from our TestResponse
					resAssert := resp.Assert(t, true)

					// 3. Test all the different HTMLAssertFunc options
					resAssert.AssertHTML(
						djester.HasElement("nav.main-menu"),
						djester.DoesNotHaveElement(".error-message"),
						djester.HasText("h1.page-title", "Dashboard"),
						djester.HasText("title", "Test Dashboard"),
						djester.HasAttribute("form#login-form", "method", "POST"),
						djester.HasAttribute("h1.page-title", "data-custom", "title-data"),
						djester.HasElementCount("ul > li", 3),
					)
				},
			},
			&djester.BasicTest{
				Label: "POST body works",
				Function: func(d *djester.Tester, t *testing.T) {
					assert := d.Assert(t, true)
					body := bytes.NewBufferString("hello")
					resp, err := d.Post("/echo", nil, nil, body)
					assert.Assert(err == nil, "POST failed: %v", err)
					defer resp.Body.Close()
					b, _ := io.ReadAll(resp.Body)
					assert.AssertEqual("hello", string(b))
				},
			},
			&djester.BasicTest{
				Label: "JSON POST works",
				Function: func(d *djester.Tester, t *testing.T) {
					assert := d.Assert(t, true)
					payload := map[string]string{"name": "test"}
					var out map[string]string
					resp, err := d.PostJson("/json", nil, nil, payload, &out)
					assert.Assert(err == nil, "JSON POST failed: %v", err)
					defer resp.Body.Close()
					assert.AssertEqual("test", out["name"])
				},
			},
			&djester.BasicTest{
				Label: "Form POST works",
				Function: func(d *djester.Tester, t *testing.T) {
					assert := d.Assert(t, true)
					form := map[string]interface{}{"foo": "bar"}
					resp, err := d.PostForm("/form", nil, nil, form)
					assert.Assert(err == nil, "Form POST failed: %v", err)
					defer resp.Body.Close()
					b, _ := io.ReadAll(resp.Body)
					assert.AssertEqual("bar", string(b))
				},
			},
			&djester.BasicTest{
				Label: "File upload works",
				Function: func(d *djester.Tester, t *testing.T) {
					assert := d.Assert(t, true)
					f, err := os.CreateTemp("", "file.txt")
					assert.Assert(err == nil, "tmp file failed: %v", err)
					defer os.Remove(f.Name())
					f.WriteString("uploadme")
					f.Seek(0, io.SeekStart)

					file, _ := os.Open(f.Name())
					defer file.Close()

					resp, err := d.PostFile("/upload", nil, nil, "file.txt", file)
					assert.Assert(err == nil, "upload failed: %v", err)
					defer resp.Body.Close()
					b, _ := io.ReadAll(resp.Body)
					assert.AssertEqual("uploadme", string(b))
				},
			},
			&djester.BasicTest{
				Label: "Failing setup triggers error",
				Function: func(d *djester.Tester, t *testing.T) {
					broken := &djester.Tester{
						BeforeSetup: func(*djester.Tester) error {
							return fmt.Errorf("intentional fail")
						},
					}
					err := broken.Setup()
					if err == nil || !strings.Contains(err.Error(), "intentional") {
						t.Fatalf("expected setup fail, got %v", err)
					}
				},
			},
		},
	}

	d.Test(t)
}

func BenchmarkDjester(b *testing.B) {
	d := &djester.Tester{
		Apps: []djester.AppInitFuncOrAppConfig{newApp},
		Tests: []djester.Test{
			&djester.BasicTest{
				Label: "Benchmark /ping",
				Benchmark: func(d *djester.Tester, b *testing.B) {
					for i := 0; i < b.N; i++ {
						resp, err := d.Get("/ping", nil, url.Values{})
						if err != nil {
							b.Fatalf("GET /ping failed: %v", err)
						}
						io.ReadAll(resp.Body)
						resp.Body.Close()
					}
				},
			},
		},
	}
	d.Bench(b)
}
