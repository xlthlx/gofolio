package web

import (
	"net/http"

	"gofolio/internal/store"
)

// knownSiteKeys whitelists editable keys, keeping arbitrary DB writes out.
var knownSiteKeys = []string{"site_title", "tagline"}

// SettingsData feeds the settings page and the site settings section.
type SettingsData struct {
	Values map[string]string
	Error  string
}

// settings hoists the store construction, matching posts()/pages().
func (h Admin) settings() store.SettingStore {
	return store.SettingStore{DB: h.DB}
}

// settingsData reads every stored setting for the templates.
func (h Admin) settingsData() SettingsData {
	values, err := h.settings().All()
	if err != nil {
		return SettingsData{}
	}
	return SettingsData{Values: values}
}

// renderSettings renders the settings page, which is what the settings route
// displays: the site settings section plus the password form.
func (h Admin) renderSettings(w http.ResponseWriter, data SettingsData) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := h.Tpl.SettingsPage(w, data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

// SiteSettingsPost saves the whitelisted keys and answers with the re-rendered
// site settings section for HTMX, so the values appear in place on the settings
// page. Plain form posts get a redirect to that page instead, which is the same
// outcome without JavaScript. The fragment is returned with a 200 because HTMX
// only swaps successful responses by default.
func (h Admin) SiteSettingsPost(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	for _, key := range knownSiteKeys {
		if v := r.PostFormValue(key); v != "" {
			if err := h.settings().Set(key, v); err != nil {
				h.siteSettingsFailure(w, r, err)
				return
			}
		}
	}

	if isHTMX(r) {
		h.renderSiteSettings(w, h.settingsData())
		return
	}
	http.Redirect(w, r, "/admin/settings/", http.StatusSeeOther)
}

// siteSettingsFailure reports a failed save. HTMX gets the section with the
// message inside it, and everything else gets a conventional error.
func (h Admin) siteSettingsFailure(w http.ResponseWriter, r *http.Request, err error) {
	if !isHTMX(r) {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	data := h.settingsData()
	data.Error = "Could not save the settings: " + err.Error()
	h.renderSiteSettings(w, data)
}

func (h Admin) renderSiteSettings(w http.ResponseWriter, data SettingsData) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := h.Tpl.SiteSettings(w, data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}
