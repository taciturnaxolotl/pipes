package sources

import (
	"context"
	"fmt"
	"time"

	"github.com/mmcdole/gofeed"

	"github.com/kierank/pipes/nodes"
)

type RSSSourceNode struct{}

func (n *RSSSourceNode) Type() string        { return "rss-source" }
func (n *RSSSourceNode) Label() string       { return "RSS Feed" }
func (n *RSSSourceNode) Description() string { return "Fetch items from an RSS or Atom feed" }
func (n *RSSSourceNode) Category() string    { return "source" }
func (n *RSSSourceNode) Inputs() int         { return 0 }
func (n *RSSSourceNode) Outputs() int        { return 1 }

func (n *RSSSourceNode) Execute(ctx context.Context, config map[string]interface{}, inputs [][]interface{}, execCtx *nodes.Context) ([]interface{}, error) {
	url, ok := config["url"].(string)
	if !ok || url == "" {
		return nil, fmt.Errorf("url is required")
	}

	execCtx.Log("rss-source", "info", fmt.Sprintf("Fetching %s", url))

	// Parse feed
	fp := gofeed.NewParser()
	feed, err := fp.ParseURLWithContext(url, ctx)
	if err != nil {
		return nil, fmt.Errorf("parse feed: %w", err)
	}

	// Convert feed items to generic interface{} slices
	var items []interface{}
	for _, item := range feed.Items {
		// Flatten author field - extract name if it's a Person struct
		var author string
		if item.Author != nil {
			author = item.Author.Name
		}

		// Parse dates to Unix timestamps for proper sorting
		var publishedAt int64
		var updatedAt int64
		if item.PublishedParsed != nil {
			publishedAt = item.PublishedParsed.Unix()
		} else if item.Published != "" {
			if t, err := parseDate(item.Published); err == nil {
				publishedAt = t.Unix()
			}
		}
		if item.UpdatedParsed != nil {
			updatedAt = item.UpdatedParsed.Unix()
		} else if item.Updated != "" {
			if t, err := parseDate(item.Updated); err == nil {
				updatedAt = t.Unix()
			}
		}

		// Extract content - prefer Content over Description
		content := item.Description
		if item.Content != "" {
			content = item.Content
		}

		// Build enclosures array (for media like images, audio, video)
		var enclosures []map[string]interface{}
		if len(item.Enclosures) > 0 {
			for _, enc := range item.Enclosures {
				enclosures = append(enclosures, map[string]interface{}{
					"url":    enc.URL,
					"type":   enc.Type,
					"length": enc.Length,
				})
			}
		}

		// Extract image URL if available
		var imageURL string
		if item.Image != nil {
			imageURL = item.Image.URL
		}

		items = append(items, map[string]interface{}{
			"title":        item.Title,
			"description":  item.Description,
			"content":      content,
			"link":         item.Link,
			"author":       author,
			"published":    item.Published,
			"published_at": publishedAt,
			"updated":      item.Updated,
			"updated_at":   updatedAt,
			"guid":         item.GUID,
			"categories":   item.Categories,
			"enclosures":   enclosures,
			"image":        imageURL,
		})
	}

	// Apply limit if specified
	if limit, ok := config["limit"].(float64); ok && limit > 0 {
		if int(limit) < len(items) {
			items = items[:int(limit)]
		}
	}

	execCtx.Log("rss-source", "info", fmt.Sprintf("Retrieved %d items", len(items)))

	return items, nil
}

func (n *RSSSourceNode) ValidateConfig(config map[string]interface{}) error {
	url, ok := config["url"].(string)
	if !ok || url == "" {
		return fmt.Errorf("url is required")
	}

	return nil
}

func (n *RSSSourceNode) GetConfigSchema() *nodes.ConfigSchema {
	return &nodes.ConfigSchema{
		Fields: []nodes.ConfigField{
			{
				Name:        "url",
				Label:       "Feed URL",
				Type:        "url",
				Required:    true,
				Placeholder: "https://example.com/feed.xml",
				HelpText:    "URL of the RSS or Atom feed",
			},
			{
				Name:         "limit",
				Label:        "Item Limit",
				Type:         "number",
				Required:     false,
				DefaultValue: 50,
				HelpText:     "Maximum number of items to fetch",
			},
		},
	}
}

// parseDate tries multiple date formats
func parseDate(s string) (time.Time, error) {
	formats := []string{
		time.RFC1123Z,
		time.RFC1123,
		time.RFC3339,
		time.RFC822Z,
		time.RFC822,
		"Mon, 2 Jan 2006 15:04:05 MST",
		"Mon, 2 Jan 2006 15:04:05 -0700",
		"2006-01-02T15:04:05Z",
		"2006-01-02T15:04:05-07:00",
		"2006-01-02 15:04:05",
		"2006-01-02",
	}

	for _, format := range formats {
		if t, err := time.Parse(format, s); err == nil {
			return t, nil
		}
	}

	return time.Time{}, fmt.Errorf("unable to parse date: %s", s)
}
