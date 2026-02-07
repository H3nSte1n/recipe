package urlparser

import (
	"fmt"
	"github.com/PuerkitoBio/goquery"
	"go.uber.org/zap"
	"strings"
)

type ContentParser interface {
	Parse(content string) (string, error)
}

type contentParser struct {
	logger *zap.Logger
}

func NewContentParser(logger *zap.Logger) ContentParser {
	return &contentParser{
		logger: logger,
	}
}

func (p *contentParser) Parse(htmlContent string) (string, error) {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(htmlContent))
	if err != nil {
		return "", fmt.Errorf("failed to parse HTML: %w", err)
	}

	if schema := extractJsonLD(doc); schema != "" {
		return schema, nil
	}

	p.cleanDocument(doc)

	content := p.extractContent(doc)

	if content == "" {
		return "", fmt.Errorf("no content found")
	}

	return content, nil
}

func (p *contentParser) cleanDocument(doc *goquery.Document) {
	doc.Find(strings.Join(UnwantedSelectors, ", ")).Remove()

	doc.Find("*").Each(func(_ int, s *goquery.Selection) {
		if strings.TrimSpace(s.Text()) == "" {
			s.Remove()
		}
	})
}

func (p *contentParser) extractContent(doc *goquery.Document) string {
	var content strings.Builder

	mainContent := doc.Find(strings.Join(MainContentSelectors, ", ")).First()

	if mainContent.Length() > 0 {
		mainContent.Find("*").Each(func(_ int, s *goquery.Selection) {
			text := strings.TrimSpace(s.Text())
			if text != "" {
				if content.Len() > 0 {
					content.WriteString("\n\n")
				}
				content.WriteString(text)
			}
		})
	} else {
		doc.Find("body").Find("*").Each(func(_ int, s *goquery.Selection) {
			text := strings.TrimSpace(s.Text())
			if text != "" {
				if content.Len() > 0 {
					content.WriteString("\n\n")
				}
				content.WriteString(text)
			}
		})
	}

	return p.cleanExtractedContent(content.String())
}

func (p *contentParser) cleanExtractedContent(content string) string {
	content = cleanText(content)

	content = strings.ReplaceAll(content, "\n\n\n", "\n\n")

	var cleaned []string
	for _, line := range strings.Split(content, "\n") {
		if len(line) > 10 {
			cleaned = append(cleaned, line)
		}
	}

	return strings.Join(cleaned, "\n")
}
