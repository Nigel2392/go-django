package images_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/Nigel2392/go-django/queries/src/quest"
	django "github.com/Nigel2392/go-django/src"
	"github.com/Nigel2392/go-django/src/contrib/editor/features/images"
	images_app "github.com/Nigel2392/go-django/src/contrib/images"
	"github.com/Nigel2392/go-django/src/core/attrs"
	"github.com/Nigel2392/go-django/src/core/filesystem/mediafiles/memory"
	"github.com/Nigel2392/go-django/src/core/logger"
	"github.com/Nigel2392/go-django/src/djester/testdb"
	"github.com/Nigel2392/go-django/src/permissions"
)

var _ permissions.PermissionTester = (*permissionBackend)(nil)

type permissionBackend struct {
}

func (p *permissionBackend) HasObjectPermission(r *http.Request, obj interface{}, perms ...string) bool {
	return true
}

func (p *permissionBackend) HasPermission(r *http.Request, perm ...string) bool {
	return true
}

func TestMain(m *testing.M) {

	var _, sqlDB = testdb.Open()
	var appFileBackend = memory.NewBackend(5)

	permissions.Tester = &permissionBackend{}

	attrs.RegisterModel(&images_app.Image{})

	if django.Global == nil {
		django.App(django.Configure(map[string]interface{}{
			django.APPVAR_DATABASE: sqlDB,
		}),
			django.Flag(
				django.FlagSkipCmds,
				django.FlagSkipChecks,
				django.FlagSkipDepsCheck,
			),
			django.Apps(
				images_app.NewAppConfig(&images_app.Options{
					MediaBackend:    appFileBackend,
					MaxByteSize:     1024 * 4,
					AllowedFileExts: []string{".jpg", ".jpeg", ".png", ".gif", ".svg"},
				}),
				images.NewAppConfig,
			),
		)

		logger.Setup(&logger.Logger{
			Level:       logger.DBG,
			WrapPrefix:  logger.ColoredLogWrapper,
			OutputDebug: os.Stdout,
			OutputInfo:  os.Stdout,
			OutputWarn:  os.Stdout,
			OutputError: os.Stderr,
		})
	}

	var t = quest.Table[*testing.T](nil,
		&images_app.Image{},
	)

	t.Create()

	if err := django.Global.Initialize(); err != nil {
		panic(fmt.Sprintf("Error initializing GO-Django: %s", err))
	}

	exitCode := m.Run()

	t.Drop()

	os.Exit(exitCode)

}

var testImage = []byte(`<svg xmlns="http://www.w3.org/2000/svg" width="100" height="100">
	<circle cx="50" cy="50" r="40" stroke="black" stroke-width="3" fill="red" />
</svg>`)

var testImage2 = []byte(`<svg xmlns="http://www.w3.org/2000/svg" width="100" height="100">
	<circle cx="50" cy="50" r="40" stroke="black" stroke-width="3" fill="blue" />
</svg>`)

//var testImage3 = []byte(`<svg xmlns="http://www.w3.org/2000/svg" width="100" height="100">
//	<circle cx="50" cy="50" r="40" stroke="black" stroke-width="3" fill="green" />
//</svg>`)

type dummyUser struct {
	IsAdministrator bool
}

func (u *dummyUser) IsAuthenticated() bool {
	return true
}

func (u *dummyUser) IsAdmin() bool {
	return u.IsAdministrator
}

func makeRequest(method, url string, filename string, image []byte) *http.Request {
	req, _ := http.NewRequest(method, url, nil)
	var b = new(bytes.Buffer)
	var w = multipart.NewWriter(b)
	defer w.Close()

	var fw io.Writer
	fw, _ = w.CreateFormFile("file", filename)
	fw.Write(image)

	req.Header.Set("Content-Type", w.FormDataContentType())
	req.Body = io.NopCloser(b)
	return req
}

type uploadResponse struct {
	Status     string `json:"status"`
	Message    string `json:"message"`
	Csrf_token string `json:"csrf_token"`
	FilePath   string `json:"filePath"`
}

func TestViews(t *testing.T) {
	images.ImageFeature.OnRegister(django.Global.Mux)
	var server = httptest.NewServer(django.Global.Mux)
	defer server.Close()

	var (
		uploadUrl = django.Reverse("upload-image")
		client    = server.Client()
	)

	t.Run("TestUploadImage", func(t *testing.T) {
		var req = makeRequest("POST", fmt.Sprintf("%s%s", server.URL, uploadUrl), "test.svg", testImage)
		var resp, err = client.Do(req)
		if err != nil {
			t.Fatalf("Expected no error, got %s", err)
		}

		if resp.StatusCode != http.StatusOK {
			var buf = new(bytes.Buffer)
			io.Copy(buf, resp.Body)
			t.Fatalf("Expected status 200, got %d, %s", resp.StatusCode, buf.String())
		}

		defer resp.Body.Close()

		var body = new(uploadResponse)
		json.NewDecoder(resp.Body).Decode(body)

		if body.Status != "success" {
			t.Fatalf("Expected status 'success', got %s, %s", body.Status, body.Message)
		}

		if body.FilePath == "" {
			t.Fatal("Expected file path, got empty")
		}

		t.Logf("Uploaded file: %s", body.FilePath)
	})

	t.Run("TestViewImage", func(t *testing.T) {
		var req = makeRequest("POST", fmt.Sprintf("%s%s", server.URL, uploadUrl), "test2.svg", testImage2)
		var resp, err = client.Do(req)
		if err != nil {
			t.Fatalf("Expected no error, got %s", err)
		}

		if resp.StatusCode != http.StatusOK {
			var buf = new(bytes.Buffer)
			io.Copy(buf, resp.Body)
			t.Fatalf("Expected status 200, got %d, %s", resp.StatusCode, buf.String())
		}

		defer resp.Body.Close()

		var body = new(uploadResponse)
		json.NewDecoder(resp.Body).Decode(body)

		if body.Status != "success" {
			t.Fatalf("Expected status 'success', got %s", body.Status)
		}

		if body.FilePath == "" {
			t.Fatal("Expected file path, got empty")
		}

		t.Logf("Uploaded file: %s", body.FilePath)

		var imageViewUrl = django.Reverse("images:serve", body.FilePath)
		var viewReq, _ = http.NewRequest(
			"GET",
			fmt.Sprintf("%s%s", server.URL, imageViewUrl),
			nil,
		)
		var viewResp, _ = client.Do(viewReq)
		if viewResp.StatusCode != http.StatusOK {
			var buf = new(bytes.Buffer)
			io.Copy(buf, viewResp.Body)
			t.Fatalf(
				"Expected status 200, got %d, %s, %s",
				viewResp.StatusCode, buf.String(), body.FilePath,
			)
		}

		defer viewResp.Body.Close()

		var buf = new(bytes.Buffer)
		io.Copy(buf, viewResp.Body)

		if !bytes.Equal(buf.Bytes(), testImage2) {
			t.Fatal("Expected image content to match")
		}
	})
}
