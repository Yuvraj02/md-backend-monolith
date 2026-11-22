package httpserver

import (
	"encoding/json"
	"time"

	articlemodels "github.com/marketing-digest/md-backend-monolith/internal/app/article/models"
	blogmodels "github.com/marketing-digest/md-backend-monolith/internal/app/blog/models"
)

// JSON shapes match the frontend TypeScript types (camelCase).

func blogJSON(b *blogmodels.Blog) map[string]any {
	if b == nil {
		return nil
	}
	stats := map[string]any{
		"totalViews":              b.TotalViews,
		"articleCount":            b.ArticleCount,
		"totalReadingTimeMinutes": b.TotalReadingTimeMinutes,
	}
	out := map[string]any{
		"id": b.ID, "name": b.Name, "slug": b.Slug, "description": b.Description,
		"ownerId": b.OwnerID, "upvotes": b.Upvotes, "stats": stats,
		"createdAt": formatTime(b.CreatedAt), "updatedAt": formatTime(b.UpdatedAt),
	}
	if b.CoverImage != "" {
		out["coverImage"] = b.CoverImage
	}
	return out
}

func articleJSON(a *articlemodels.Article) map[string]any {
	if a == nil {
		return nil
	}
	var content any = []any{}
	if a.ContentJSON != "" {
		_ = json.Unmarshal([]byte(a.ContentJSON), &content)
	}
	media := map[string]any{}
	if a.ImageURL != "" {
		media["imageUrl"] = a.ImageURL
	}
	if a.VideoURL != "" {
		media["videoUrl"] = a.VideoURL
	}
	tags := decodeTags(a.TagsJSON)
	comments := make([]any, 0, len(a.Comments))
	for i := range a.Comments {
		comments = append(comments, commentJSON(&a.Comments[i]))
	}
	out := map[string]any{
		"id": a.ID, "blogId": a.BlogID, "title": a.Title, "excerpt": a.Excerpt,
		"media": media, "tags": tags, "content": content, "status": string(a.Status),
		"upvotes": a.Upvotes, "views": a.Views, "viewsThisWeek": a.ViewsThisWeek,
		"reads": a.Reads, "readingTimeMinutes": a.ReadingTimeMinutes,
		"comments": comments, "createdAt": formatTime(a.CreatedAt), "updatedAt": formatTime(a.UpdatedAt),
		"hasDraft": a.HasDraft,
	}
	if a.PublishedAt != nil {
		out["publishedAt"] = formatTime(*a.PublishedAt)
	}
	if a.LastSaved != nil {
		out["lastSaved"] = formatTime(*a.LastSaved)
	}
	if tags == nil {
		out["tags"] = []string{}
	}
	return out
}

func commentJSON(c *articlemodels.Comment) map[string]any {
	if c == nil {
		return nil
	}
	out := map[string]any{
		"id": c.ID, "articleId": c.ArticleID, "name": c.Name,
		"email": c.Email, "body": c.Body, "createdAt": formatTime(c.CreatedAt),
	}
	if c.Website != "" {
		out["website"] = c.Website
	}
	return out
}

func draftJSON(blogID string, d *articlemodels.Draft) map[string]any {
	if d == nil {
		return nil
	}
	var content any = []any{}
	if d.ContentJSON != "" {
		_ = json.Unmarshal([]byte(d.ContentJSON), &content)
	}
	media := map[string]any{}
	if d.ImageURL != "" {
		media["imageUrl"] = d.ImageURL
	}
	if d.VideoURL != "" {
		media["videoUrl"] = d.VideoURL
	}
	tags := decodeTags(d.TagsJSON)
	if tags == nil {
		tags = []string{}
	}
	return map[string]any{
		"id": d.ID, "articleId": d.ArticleID, "blogId": blogID,
		"title": d.Title, "excerpt": d.Excerpt, "media": media, "tags": tags,
		"content": content, "readingTimeMinutes": d.ReadingTimeMinutes,
		"createdAt": formatTime(d.CreatedAt), "updatedAt": formatTime(d.UpdatedAt),
	}
}

func decodeTags(raw string) []string {
	if raw == "" {
		return nil
	}
	var tags []string
	_ = json.Unmarshal([]byte(raw), &tags)
	return tags
}

func formatTime(t time.Time) string {
	return t.UTC().Format(time.RFC3339)
}