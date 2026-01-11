package sources

import (
	"context"
	"fmt"

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
		
		items = append(items, map[string]interface{}{
			"title":       item.Title,
			"description": item.Description,
			"link":        item.Link,
			"author":      author,
			"published":   item.Published,
			"updated":     item.Updated,
			"guid":        item.GUID,
			"categories":  item.Categories,
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
