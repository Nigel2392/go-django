package main

import (
	"fmt"

	"github.com/Nigel2392/go-django/core/app/tool"
	"github.com/Nigel2392/go-django/core/flag"
	"github.com/google/uuid"
)

func main() {
	var flags = flag.NewFlags("Go-Django-Tools")
	flags.Info = `Go-Django-Tools is a tool to help you create a new Go-Django project.`
	flags.Register("env", "", "", func(value flag.Value) error {
		if !value.IsZero() {
			var env_str_generated_secret = fmt.Sprintf(tool.Env_template, uuid.New().String())
			return tool.CreateFileStr(value.String(), env_str_generated_secret)
		}
		return nil
	})

	flags.Register("start", "", "", tool.StartProject)

	if !flags.Run() {
		fmt.Println("No flags passed")
	}
}
