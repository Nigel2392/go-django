package openauth2

import (
	"net/http"

	"github.com/Nigel2392/go-django/src/core/trans"
)

type boundProviderInfo struct {
	r   *http.Request
	cnf ConfigInfo
}

func (oa *boundProviderInfo) Provider() string {
	return oa.cnf.Provider
}

func (oa *boundProviderInfo) ProviderLabel() string {
	label, ok := trans.GetText(oa.r.Context(), oa.cnf.ProviderLabel)
	if ok {
		return label
	}

	return oa.cnf.Provider
}

func (oa *boundProviderInfo) ProviderLogoURL() string {
	if oa.cnf.ProviderLogoURL == nil {
		return ""
	}
	return oa.cnf.ProviderLogoURL(oa.r)
}

func (oa *boundProviderInfo) DocumentationURL() string {
	if oa.cnf.DocumentationURL == nil {
		return ""
	}
	return oa.cnf.DocumentationURL(oa.r)
}

func (oa *boundProviderInfo) PrivacyPolicyURL() string {
	if oa.cnf.PrivacyPolicyURL == nil {
		return ""
	}
	return oa.cnf.PrivacyPolicyURL(oa.r)
}
