package web_test

import (
	"testing"

	web "gofolio/internal/handler/http"
)

func TestSlugify(t *testing.T) {
	cases := map[string]string{
		"Hello":             "hello",
		"Launch Day":        "launch-day",
		"Hello, World!":     "hello-world",
		"  spaced  out ":    "spaced-out",
		"...":               "",
		"Città":             "citta",
		"Perché l'Economia": "perche-l-economia",
		"façade & naïve":    "facade-naive",
		"ÀÈÌÒÙ":             "aeiou",
		"Café-Core":         "cafe-core",
		"under_score.dot":   "under-score-dot",
	}
	for in, want := range cases {
		if got := web.Slugify(in); got != want {
			t.Errorf("Slugify(%q) = %q, want %q", in, got, want)
		}
	}
}
