package internal

import (
	"strings"

	"github.com/PuerkitoBio/goquery"
)

type Cleaner struct{}

func (c *Cleaner) CleanHTML(htmlContent string) string {
	reader := strings.NewReader(htmlContent)

	doc, err := goquery.NewDocumentFromReader(reader)
	if err != nil {
		return htmlContent
	}

	doc.Find("script, style, nav, footer, header, iframe, noscript").Remove()

	var content string
	jobSection := doc.Find("div.job-description, section.job-details, #job-content")

	if jobSection.Length() > 0 {
		content, _ = jobSection.Html()
	} else {
		body := doc.Find("body")
		body.Find("script, style, nav, footer, header, iframe").Remove()
		content, _ = body.Html()
	}

	content = strings.TrimSpace(content)
	content = strings.ReplaceAll(content, "\n\n\n", "\n\n")

	return content
}

func (c *Cleaner) CleanLLMResponse(response string) string {
	if strings.Contains(response, "```json") && strings.Contains(response, "```") {
		start := strings.Index(response, "```json")
		if start == -1 {
			start = strings.Index(response, "```")
		} else {
			start += 7 // Length of "```json"
		}

		end := strings.LastIndex(response, "```")
		if start != -1 && end != -1 && end > start {
			response = response[start:end]
		}
	}

	response = strings.TrimSpace(response)

	return response
}

func NewCleaner() *Cleaner {
	return &Cleaner{}
}
