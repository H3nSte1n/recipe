package urlparser

import (
	"fmt"
	"github.com/PuerkitoBio/goquery"
	"net/http"
	"strings"
)

func defaultRedirectPolicy(req *http.Request, via []*http.Request) error {
	if len(via) >= 10 {
		return fmt.Errorf("too many redirects")
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
