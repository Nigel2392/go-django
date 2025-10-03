package formsapp

import (
	"net/http"

	"github.com/Nigel2392/go-django/src/forms"
	"github.com/Nigel2392/go-django/src/forms/fields"
	"github.com/Nigel2392/go-django/src/forms/widgets"
)

func NewContactForm(request *http.Request) forms.Form {

	// Fields to add to the form
	//
	// You can use the [fields.Field] interface to create custom fields
	var fields = []fields.Field{
		fields.CharField(
			fields.Name("name"),
			fields.Label("Name"),
			fields.Required(true),
			fields.MinLength(2),
			fields.MaxLength(100),
			fields.ReadOnly(false),
			fields.Placeholder("Enter your full name"),
			fields.HelpText("Please enter your full name."),
			fields.Default("John Doe"),
		),
		fields.EmailField(
			fields.Name("email"),
			fields.Label("Email"),
			fields.Required(true),
			fields.ReadOnly(false),
			fields.Placeholder("Enter your email address"),
			fields.HelpText("Please enter a valid email address."),
			fields.MinLength(5),
			fields.MaxLength(255),
			fields.Default("example@example.com"),
		),
		fields.CharField(
			fields.Name("subject"),
			fields.Label("Subject"),
			fields.Required(true),
			fields.MinLength(2),
			fields.MaxLength(100),
			fields.ReadOnly(false),
			fields.Placeholder("Enter the subject of your message"),
			fields.HelpText("Please enter the subject of your message."),
			fields.Attributes(map[string]string{"autocomplete": "off"}),
			fields.Default("Hello"),
		),
		fields.CharField(
			fields.Name("message"),
			fields.Label("Message"),
			fields.Required(true),
			fields.MinLength(10),
			fields.MaxLength(5000),
			fields.ReadOnly(false),
			fields.Placeholder("Enter your message"),
			fields.HelpText("Please enter your message."),
			fields.Attributes(map[string]string{"autocomplete": "off"}),
			fields.Default("I would like to get in touch with you."),

			// Pass a custom widget to the field to use a textarea instead of a text input
			fields.Widget(widgets.NewTextarea(nil)),
		),
	}

	var form = forms.NewBaseForm(
		request.Context(),
		// Either directly pass the initialization options here
		// or see the [forms.Initialize] function below.
		forms.WithRequestData(
			// If the request method is not POST, we will not parse the form data.
			// This is useful for GET requests where we want to display the form without processing it
			http.MethodPost,

			// Pass the request
			// This is where the form will get its data from
			request,
		),
	)

	return forms.Initialize(
		form,

		// or pass initialization options here, this is useful
		// when you have not initialized the form yourself.

		// Add the fields to the form
		forms.WithFields(fields...),
	)
}
