package widgets

func NewTextInput(attrs map[string]string) Widget {
	return NewBaseWidget("text", "forms/widgets/text.html", attrs)
}

func NewTextarea(attrs map[string]string) Widget {
	return NewBaseWidget("textarea", "forms/widgets/textarea.html", attrs)
}

func NewEmailInput(attrs map[string]string) Widget {
	return NewBaseWidget("email", "forms/widgets/email.html", attrs)
}

func NewPasswordInput(attrs map[string]string) Widget {
	return NewBaseWidget("password", "forms/widgets/password.html", attrs)
}

func NewHiddenInput(attrs map[string]string) Widget {
	return NewBaseWidget("hidden", "forms/widgets/hidden.html", attrs)
}
