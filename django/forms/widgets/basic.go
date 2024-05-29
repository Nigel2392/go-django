package widgets

func NewTextInput(attrs map[string]string) Widget {
	return NewBaseWidget(S("text"), "widgets/text.html", attrs)
}

func NewTextarea(attrs map[string]string) Widget {
	return NewBaseWidget(S("textarea"), "widgets/textarea.html", attrs)
}

func NewEmailInput(attrs map[string]string) Widget {
	return NewBaseWidget(S("email"), "widgets/email.html", attrs)
}

func NewPasswordInput(attrs map[string]string) Widget {
	return NewBaseWidget(S("password"), "widgets/password.html", attrs)
}

func NewHiddenInput(attrs map[string]string) Widget {
	return NewBaseWidget(S("hidden"), "widgets/hidden.html", attrs)
}
