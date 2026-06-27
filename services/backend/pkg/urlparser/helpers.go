package urlparser

import (
	"fmt"
	"github.com/PuerkitoBio/goquery"
	"net/http"
	"strings"
)

// defaultRedirectPolicy bounds the redirect chain and rejects any hop that is
// not http/https. The resolved-IP check for each hop is enforced authoritatively
// by safeDialContext, which runs on every dial including redirects.
func defaultRedirectPolicy(req *http.Request, via []*http.Request) error {
	if len(via) >= 10 {
		return fmt.Errorf("too many redirects")
	}
	if req.URL.Scheme != "http" && req.URL.Scheme != "https" {
		return fmt.Errorf("redirect to unsupported scheme %q", req.URL.Scheme)
	}
	return nil
}

func extractJsonLD(doc *goquery.Document) string {
	var recipeData string
	doc.Find("script[type='application/ld+json']").Each(func(_ int, script *goquery.Selection) {
		jsonContent := script.Text()
		if strings.Contains(strings.ToLower(jsonContent), "recipe") {
			recipeData = jsonContent
			return
		}
	})
	return recipeData
}

func cleanText(text string) string {
	text = strings.Join(strings.Fields(text), " ")

	unwantedPhrases := []string{
		"advertisement",
		"subscribe to our newsletter",
		"share this recipe",
		"print recipe",
		"save recipe",
	}

	for _, phrase := range unwantedPhrases {
		text = strings.ReplaceAll(strings.ToLower(text), phrase, "")
	}

	return strings.TrimSpace(text)
}
