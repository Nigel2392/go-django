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
	"github.com/pkg/errors"
	"google.golang.org/protobuf/proto"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "error generating code: %v\n", err)
		os.Exit(2)
	}
}

func Generate(ctx context.Context, req *plugin.GenerateRequest) (*plugin.GenerateResponse, error) {

	var opts = new(codegen.CodeGeneratorOptions)
	if err := json.Unmarshal(req.PluginOptions, opts); err != nil {
		return nil, errors.Wrap(err, "failed to unmarshal options into CodeGeneratorOptions")
	}

	var generator, err = codegen.New(req, opts)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create code generator")
	}

	var files = make([]*plugin.File, 0)
	for _, schema := range req.Catalog.Schemas {

		var tmplObj = generator.BuildTemplateObject(schema)
		var b = new(bytes.Buffer)
		if err := generator.Render(b, codegen.GenerateDefinerTemplate, tmplObj); err != nil {
			return nil, errors.Wrap(err, "failed to render template")
		}

		var file = &plugin.File{
			Name:     opts.OutFile,
			Contents: b.Bytes(),
		}

		files = append(files, file)
	}

	return &plugin.GenerateResponse{Files: files}, nil
}

func run() error {
	var req plugin.GenerateRequest
	reqBlob, err := io.ReadAll(os.Stdin)
	if err != nil {
		return errors.Wrap(
			err, "failed to read request",
		)
	}
	if err := proto.Unmarshal(reqBlob, &req); err != nil {
		return errors.Wrap(
			err, "failed to unmarshal request",
		)
	}

	resp, err := Generate(context.Background(), &req)
	if err != nil {
		return errors.Wrap(
			err, "failed to generate response",
		)
	}

	respBlob, err := proto.Marshal(resp)
	if err != nil {
		return errors.Wrap(
			err, "failed to marshal response",
		)
	}
	w := bufio.NewWriter(os.Stdout)
	if _, err := w.Write(respBlob); err != nil {
		return errors.Wrap(
			err, "failed to write response",
		)
	}
	if err := w.Flush(); err != nil {
		return errors.Wrap(
			err, "failed to flush response",
		)
	}
	return nil
}
