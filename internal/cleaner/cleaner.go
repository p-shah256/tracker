package cleaner

import (
	"regexp"
	"strings"

	"github.com/PuerkitoBio/goquery"
)

type Cleaner struct{}

func NewCleaner() *Cleaner {
	return &Cleaner{}
}

func (c *Cleaner) CleanHTML(html string) string {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	if err != nil {
		return stripTags(html)
	}
	doc.Find("script, style, nav, header, footer, iframe, noscript").Remove()
	doc.Find(".menu, .navigation, .social, .banner, .ads, .cookie, .popup").Remove()
	doc.Find("div:empty, span:empty").Remove()
	var textBlocks []string
	doc.Find("p, li, h1, h2, h3, h4, h5, h6").Each(func(i int, s *goquery.Selection) {
		text := strings.TrimSpace(s.Text())
		if len(text) > 0 {
			textBlocks = append(textBlocks, text)
		}
	})
	if len(textBlocks) > 0 {
		return strings.Join(textBlocks, "\n\n")
	}

	bodyText := strings.TrimSpace(doc.Find("body").Text())
	if len(bodyText) > 0 {
		return cleanText(bodyText)
	}

	return cleanText(doc.Text())
}

func (c *Cleaner) CleanLlmResponse(response string) string {
	if !strings.Contains(response, "```") {
		return strings.TrimSpace(response)
	}

	start := -1
	if strings.Contains(response, "```json") {
		start = strings.Index(response, "```json") + 7
	} else if strings.Contains(response, "```yaml") {
		start = strings.Index(response, "```yaml") + 7
	} else {
		// Handle generic code blocks
		start = strings.Index(response, "```") + 3
	}

	end := strings.LastIndex(response, "```")

	if start != -1 && end != -1 && end > start {
		return strings.TrimSpace(response[start:end])
	}

	return strings.TrimSpace(response)
}

func stripTags(html string) string {
	re := regexp.MustCompile("<[^>]*>")
	text := re.ReplaceAllString(html, " ")
	return cleanText(text)
}

func cleanText(text string) string {
	re := regexp.MustCompile(`\s+`)
	text = re.ReplaceAllString(text, " ")
	return strings.TrimSpace(text)
}
