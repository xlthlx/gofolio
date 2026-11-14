package web

import (
	"database/sql"
	"errors"
	"html/template"
	"io"
	"net/http"

	"github.com/go-chi/chi/v5"

	"gofolio/internal/model"
	"gofolio/internal/store"
)

// Defaults for the site identity, used until the settings form has been filled
// in. They match the fallbacks the settings template renders.
const (
	defaultSiteTitle = "goFolio"
	defaultTagline   = "A simple Go CMS"
)

// SiteInfo is the site identity every public page shows, read from settings.
type SiteInfo struct {
	Title   string
	Tagline string
}

// SiteData is what the public templates render. The pages share one layout, so
// the site identity and the session state travel with every response, while
// Posts, Post, Page and Content carry what the page itself is about.
type SiteData struct {
	Site          SiteInfo
	Authenticated bool
	// PageTitle is set by the views that have a title of their own, so the
	// browser tab reads "Post title - Site title" rather than the site alone.
	PageTitle string
	Posts     []model.Post
	Post      *model.Post
	Page      *model.Page
	// Content is the page or post body as HTML, straight from the admin editor.
	// Escaping it would print the markup instead of rendering the article, so it
	// is the one value that deliberately bypasses escaping. Any future source of
	// public content, such as comments, would have to be sanitised first.
	Content template.HTML
}

// siteData collects the parts of every public response: the site identity from
// settings, with the fallbacks above when nothing is stored, and whether the
// visitor is signed in, which decides what the navigation offers.
func (h Public) siteData(r *http.Request) SiteData {
	info := SiteInfo{Title: defaultSiteTitle, Tagline: defaultTagline}
	if values, err := (store.SettingStore{DB: h.DB}).All(); err == nil {
		if v := values["site_title"]; v != "" {
			info.Title = v
		}
		if v := values["tagline"]; v != "" {
			info.Tagline = v
		}
	}
	return SiteData{
		Site:          info,
		Authenticated: h.Sess != nil && h.Sess.IsAuthenticated(r),
	}
}

// HomePage renders the list of published posts. Drafts never reach it, because
// the store only lists published rows.
func (h Public) HomePage(w http.ResponseWriter, r *http.Request) {
	posts, err := h.posts().ListPublished()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	data := h.siteData(r)
	data.Posts = posts
	h.renderSite(w, http.StatusOK, func(out io.Writer) error {
		return h.Tpl.HomePage(out, data)
	})
}

// PostPage renders one published post. A draft, or a slug that does not exist,
// is a 404, which is the visibility rule the JSON API applies as well.
func (h Public) PostPage(w http.ResponseWriter, r *http.Request) {
	post, err := h.posts().GetPublishedBySlug(chi.URLParam(r, "slug"))
	if errors.Is(err, sql.ErrNoRows) {
		h.notFound(w, r)
		return
	}
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	data := h.siteData(r)
	data.Post = post
	data.PageTitle = post.Title
	data.Content = template.HTML(post.Content)
	h.renderSite(w, http.StatusOK, func(out io.Writer) error {
		return h.Tpl.PostPage(out, data)
	})
}

// PagePage renders one page. Pages are reachable whatever their status says,
// which is how the JSON API has always treated them.
func (h Public) PagePage(w http.ResponseWriter, r *http.Request) {
	page, err := h.pages().GetBySlug(chi.URLParam(r, "slug"))
	if errors.Is(err, sql.ErrNoRows) {
		h.notFound(w, r)
		return
	}
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	data := h.siteData(r)
	data.Page = page
	data.PageTitle = page.Title
	data.Content = template.HTML(page.Content)
	h.renderSite(w, http.StatusOK, func(out io.Writer) error {
		return h.Tpl.PagePage(out, data)
	})
}

// notFound renders the themed 404 page. It matters more than usual here because
// HTMX boosts navigation on the public site: without an HTML body there would be
// nothing to swap in, and a stale link would appear to do nothing at all.
func (h Public) notFound(w http.ResponseWriter, r *http.Request) {
	data := h.siteData(r)
	h.renderSite(w, http.StatusNotFound, func(out io.Writer) error {
		return h.Tpl.NotFound(out, data)
	})
}

// renderSite writes the status and the content type before handing over to the
// template, so a template error cannot change a status that was already right.
func (h Public) renderSite(w http.ResponseWriter, status int, render func(io.Writer) error) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(status)
	if err := render(w); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}
