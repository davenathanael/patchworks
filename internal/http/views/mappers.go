package views

import "github.com/davenathanael/patchwork/internal/core"

func ToLinkItems(items []core.Bookmark) []LinkItem {
	result := make([]LinkItem, 0, len(items))
	for _, item := range items {
		result = append(result, LinkItem{
			ID:        item.ID.String(),
			Title:     item.Title,
			URL:       item.URL,
			Tags:      item.Tags,
			CreatedAt: item.CreatedAt,
		})
	}
	return result
}

func ToCollectionItems(items []core.Collection) []CollectionFilterItem {
	result := make([]CollectionFilterItem, 0, len(items))
	for _, item := range items {
		result = append(result, CollectionFilterItem{
			ID:    item.ID.String(),
			Name:  item.Name,
			Count: item.BookmarkCount,
		})
	}
	return result
}

func ToTagItems(items []core.Tag) []TagItem {
	result := make([]TagItem, 0, len(items))
	for _, item := range items {
		result = append(result, TagItem{
			Name:  item.Name,
			Count: item.BookmarkCount,
		})
	}
	return result
}
