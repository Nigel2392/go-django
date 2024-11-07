package main

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"

	"github.com/Nigel2392/go-django/cmd/go-django-definitions/internal/codegen"
	"github.com/Nigel2392/go-django/cmd/go-django-definitions/internal/codegen/plugin"
	"github.com/Nigel2392/go-django/cmd/go-django-definitions/internal/logger"
	"github.com/pkg/errors"
	"google.golang.org/protobuf/proto"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "error generating code: %v\n", err)
		os.Exit(2)
	}
}

func unpackJSON(b []byte) string {
	var m map[string]interface{}
	if err := json.Unmarshal(b, &m); err != nil {
		return string(b)
	}
	return fmt.Sprintf("%+v", m)
}

func Generate(ctx context.Context, req *plugin.GenerateRequest) (*plugin.GenerateResponse, error) {

	var opts = new(codegen.CodeGeneratorOptions)
	if err := json.Unmarshal(req.PluginOptions, opts); err != nil {
		return nil, errors.Wrap(err, "failed to unmarshal options into CodeGeneratorOptions")
	}

	logger.Logf("Generating code with plugin options: %s\n", req.PluginOptions)
	logger.Logf("Generating code with global options: %s\n", req.GlobalOptions)

	var generator, err = codegen.New(req, opts)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create code generator")
	}

	var files = make([]*plugin.File, 0)
	for _, schema := range req.Catalog.Schemas {

		var tmplObj = generator.BuildTemplateObject(schema)

		var templates = make([]string, 0)
		templates = append(templates, codegen.GenerateDefinerTemplate)

		if opts.GenerateModelsMethods {
			templates = append(templates, codegen.GenerateModelsTemplate)
		}

		if opts.GenerateAdminSetup {
			templates = append(templates, codegen.GenerateAdminTemplate)
		}

		for _, tmpl := range templates {
			var b = new(bytes.Buffer)
			if err := generator.Render(b, tmpl, tmplObj); err != nil {
				return nil, errors.Wrap(err, "failed to render template")
			}

			var n string
			if opts.OutFile != "" {
				n = fmt.Sprintf(
					"%s_%s", codegen.Prefixes[tmpl], opts.OutFile,
				)
			} else {
				n = fmt.Sprintf(
					"%s.go", codegen.Prefixes[tmpl],
				)
			}

			var file = &plugin.File{
				Name:     n,
				Contents: b.Bytes(),
			}

			files = append(files, file)
		}
	}

	return &plugin.GenerateResponse{Files: files}, nil
}

func run() error {
	// Read the protobuf request from stdin.
	var req plugin.GenerateRequest
	reqBlob, err := io.ReadAll(os.Stdin)
	if err != nil {
		return errors.Wrap(
			err, "failed to read request",
		)
	}

	// Unmarshal the request with protobufs
	if err := proto.Unmarshal(reqBlob, &req); err != nil {
		return errors.Wrap(
			err, "failed to unmarshal request",
		)
	}

	// Generate the response
	resp, err := Generate(context.Background(), &req)
	if err != nil {
		return errors.Wrap(
			err, "failed to generate response",
		)
	}

	// Marshal the response with protobufs
	respBlob, err := proto.Marshal(resp)
	if err != nil {
		return errors.Wrap(
			err, "failed to marshal response",
		)
	}

	// Write the response to stdout
	w := bufio.NewWriter(os.Stdout)
	if _, err := w.Write(respBlob); err != nil {
		return errors.Wrap(
			err, "failed to write response",
		)
	}

	// Flush the buffer
	if err := w.Flush(); err != nil {
		return errors.Wrap(
			err, "failed to flush response",
		)
	}
	return nil
}
