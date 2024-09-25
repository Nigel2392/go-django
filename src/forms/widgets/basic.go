package widgets

func NewTextInput(attrs map[string]string) Widget {
	return NewBaseWidget(S("text"), "forms/widgets/text.html", attrs)
}

func NewTextarea(attrs map[string]string) Widget {
	return NewBaseWidget(S("textarea"), "forms/widgets/textarea.html", attrs)
}

func NewEmailInput(attrs map[string]string) Widget {
	return NewBaseWidget(S("email"), "forms/widgets/email.html", attrs)
}

func NewPasswordInput(attrs map[string]string) Widget {
	return NewBaseWidget(S("password"), "forms/widgets/password.html", attrs)
}

func NewHiddenInput(attrs map[string]string) Widget {
	return NewBaseWidget(S("hidden"), "forms/widgets/hidden.html", attrs)
}
