package web

import (
	"net/http"

	"gofolio/internal/store"
)

// knownSiteKeys whitelists editable keys, keeping arbitrary DB writes out.
var knownSiteKeys = []string{"site_title", "tagline"}

// SettingsData feeds the site-settings page.
type SettingsData struct {
	Values map[string]string
}

// settings hoists the store construction, matching posts()/pages().
func (h Admin) settings() store.SettingStore {
	return store.SettingStore{DB: h.DB}
}

func (h Admin) settingsData() SettingsData {
	values, err := h.settings().All()
	if err != nil {
		return SettingsData{}
	}
	return SettingsData{Values: values}
}

// SiteSettingsGet renders the site-settings form with current values.
func (h Admin) SiteSettingsGet(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := h.Tpl.SiteSettings(w, h.settingsData()); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

// SiteSettingsPost accepts the form post, saves whitelisted keys, and
// re-renders the page so saved values are visible.
func (h Admin) SiteSettingsPost(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	for _, key := range knownSiteKeys {
		if v := r.PostFormValue(key); v != "" {
			if err := h.settings().Set(key, v); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
		}
	}
	h.SiteSettingsGet(w, r)
}
