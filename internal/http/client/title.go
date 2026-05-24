package client

import (
	"context"
	"io"
	"net/http"
	"net/url"
	"strings"
)

func (c *Client) FetchPageTitle(ctx context.Context, u *url.URL) string {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return u.String()
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return u.String()
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return u.String()
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return u.String()
	}

	title := extractTitle(string(body))
	if title == "" {
		return u.String()
	}

	return title
}

func extractTitle(html string) string {
	lower := strings.ToLower(html)

	titleTag := strings.Index(lower, "<title")
	if titleTag == -1 {
		return ""
	}

	closeBracket := strings.Index(lower[titleTag:], ">")
	if closeBracket == -1 {
		return ""
	}

	contentStart := titleTag + closeBracket + 1

	endTag := strings.Index(lower[contentStart:], "</title>")
	if endTag == -1 {
		return ""
	}

	return strings.TrimSpace(html[contentStart : contentStart+endTag])
}
