package httpserver

import (
	"encoding/json"
	"io"
	"net/http"

	"github.com/marketing-digest/md-backend-monolith/internal/app/article"
)

func (s *Server) handleBlogPing(w http.ResponseWriter, r *http.Request) {
	msg := "pong"
	if v := r.URL.Query().Get("message"); v != "" {
		msg = v
	}
	writeJSON(w, http.StatusOK, map[string]string{"message": msg, "service": "md-backend-monolith"})
}

func (s *Server) handleListBlogs(w http.ResponseWriter, r *http.Request) {
	list, err := s.blogs.List(r.Context())
	if err != nil {
		writeServiceError(w, err)
		return
	}
	out := make([]any, 0, len(list))
	for _, b := range list {
		out = append(out, blogJSON(b))
	}
	writeJSON(w, http.StatusOK, out)
}

func (s *Server) handleGetBlog(w http.ResponseWriter, r *http.Request) {
	b, err := s.blogs.Get(r.Context(), r.PathValue("blogID"))
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, blogJSON(b))
}

func (s *Server) handleCreateBlog(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Name        string `json:"name"`
		Description string `json:"description"`
		OwnerID     string `json:"ownerId"`
	}
	if err := json.NewDecoder(io.LimitReader(r.Body, 1<<20)).Decode(&body); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json"})
		return
	}
	b, err := s.blogs.Create(r.Context(), body.Name, body.Description, body.OwnerID)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, blogJSON(b))
}

func (s *Server) handleUpdateBlog(w http.ResponseWriter, r *http.Request) {
	var body map[string]any
	if err := json.NewDecoder(io.LimitReader(r.Body, 1<<20)).Decode(&body); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json"})
		return
	}
	var name, description, coverImage, slug *string
	if v, ok := body["name"].(string); ok {
		name = &v
	}
	if v, ok := body["description"].(string); ok {
		description = &v
	}
	if v, ok := body["coverImage"].(string); ok {
		coverImage = &v
	}
	if v, ok := body["slug"].(string); ok {
		slug = &v
	}
	b, err := s.blogs.Update(r.Context(), r.PathValue("blogID"), name, description, coverImage, slug)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, blogJSON(b))
}

func (s *Server) handleUpvoteBlog(w http.ResponseWriter, r *http.Request) {
	b, err := s.blogs.Upvote(r.Context(), r.PathValue("blogID"))
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, blogJSON(b))
}

func (s *Server) handleListArticles(w http.ResponseWriter, r *http.Request) {
	list, err := s.articles.ListByBlog(r.Context(), r.PathValue("blogID"))
	if err != nil {
		writeServiceError(w, err)
		return
	}
	out := make([]any, 0, len(list))
	for _, a := range list {
		out = append(out, articleJSON(a))
	}
	writeJSON(w, http.StatusOK, out)
}

func (s *Server) handleGetArticle(w http.ResponseWriter, r *http.Request) {
	a, err := s.articles.Get(r.Context(), r.PathValue("blogID"), r.PathValue("articleID"))
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, articleJSON(a))
}

func (s *Server) handleCreateArticle(w http.ResponseWriter, r *http.Request) {
	a, err := s.articles.Create(r.Context(), r.PathValue("blogID"))
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, articleJSON(a))
}

func (s *Server) handleUpdateArticle(w http.ResponseWriter, r *http.Request) {
	patch, ok := decodeArticlePatch(r)
	if !ok {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json"})
		return
	}
	a, err := s.articles.Update(r.Context(), r.PathValue("blogID"), r.PathValue("articleID"), patch)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, articleJSON(a))
}

func (s *Server) handleDeleteArticle(w http.ResponseWriter, r *http.Request) {
	if err := s.articles.Delete(r.Context(), r.PathValue("blogID"), r.PathValue("articleID")); err != nil {
		writeServiceError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleUpvoteArticle(w http.ResponseWriter, r *http.Request) {
	a, err := s.articles.Upvote(r.Context(), r.PathValue("blogID"), r.PathValue("articleID"))
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, articleJSON(a))
}

func (s *Server) handlePublishArticle(w http.ResponseWriter, r *http.Request) {
	a, err := s.articles.PublishArticle(r.Context(), r.PathValue("blogID"), r.PathValue("articleID"))
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, articleJSON(a))
}

func (s *Server) handleEnsureDraft(w http.ResponseWriter, r *http.Request) {
	blogID := r.PathValue("blogID")
	d, err := s.articles.EnsureDraft(r.Context(), blogID, r.PathValue("articleID"))
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, draftJSON(blogID, d))
}

func (s *Server) handleGetDraft(w http.ResponseWriter, r *http.Request) {
	blogID := r.PathValue("blogID")
	d, err := s.articles.GetDraft(r.Context(), blogID, r.PathValue("articleID"))
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, draftJSON(blogID, d))
}

func (s *Server) handleUpdateDraft(w http.ResponseWriter, r *http.Request) {
	patch, ok := decodeDraftPatch(r)
	if !ok {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json"})
		return
	}
	blogID := r.PathValue("blogID")
	d, err := s.articles.UpdateDraft(r.Context(), blogID, r.PathValue("articleID"), patch)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, draftJSON(blogID, d))
}

func (s *Server) handlePublishDraft(w http.ResponseWriter, r *http.Request) {
	a, err := s.articles.PublishDraft(r.Context(), r.PathValue("blogID"), r.PathValue("articleID"))
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, articleJSON(a))
}

func (s *Server) handleListPublished(w http.ResponseWriter, r *http.Request) {
	list, err := s.articles.ListPublished(r.Context())
	if err != nil {
		writeServiceError(w, err)
		return
	}
	out := make([]any, 0, len(list))
	for _, a := range list {
		out = append(out, articleJSON(a))
	}
	writeJSON(w, http.StatusOK, out)
}

func (s *Server) handleListTrending(w http.ResponseWriter, r *http.Request) {
	list, err := s.articles.ListTrending(r.Context())
	if err != nil {
		writeServiceError(w, err)
		return
	}
	out := make([]any, 0, len(list))
	for _, a := range list {
		out = append(out, articleJSON(a))
	}
	writeJSON(w, http.StatusOK, out)
}

func (s *Server) handleGetPublishedArticle(w http.ResponseWriter, r *http.Request) {
	a, err := s.articles.GetPublished(r.Context(), r.PathValue("articleID"))
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, articleJSON(a))
}

func (s *Server) handleListRelated(w http.ResponseWriter, r *http.Request) {
	list, err := s.articles.ListRelated(r.Context(), r.PathValue("articleID"), 3)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	out := make([]any, 0, len(list))
	for _, a := range list {
		out = append(out, articleJSON(a))
	}
	writeJSON(w, http.StatusOK, out)
}

func (s *Server) handleRecordView(w http.ResponseWriter, r *http.Request) {
	if err := s.articles.RecordView(r.Context(), r.PathValue("articleID")); err != nil {
		writeServiceError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleRecordRead(w http.ResponseWriter, r *http.Request) {
	if err := s.articles.RecordRead(r.Context(), r.PathValue("articleID")); err != nil {
		writeServiceError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleListComments(w http.ResponseWriter, r *http.Request) {
	list, err := s.articles.ListComments(r.Context(), r.PathValue("articleID"))
	if err != nil {
		writeServiceError(w, err)
		return
	}
	out := make([]any, 0, len(list))
	for _, c := range list {
		out = append(out, commentJSON(c))
	}
	writeJSON(w, http.StatusOK, out)
}

func (s *Server) handleCreateComment(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Name    string `json:"name"`
		Email   string `json:"email"`
		Website string `json:"website"`
		Body    string `json:"body"`
	}
	if err := json.NewDecoder(io.LimitReader(r.Body, 1<<20)).Decode(&body); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json"})
		return
	}
	c, err := s.articles.CreateComment(r.Context(), r.PathValue("articleID"), body.Name, body.Email, body.Website, body.Body)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, commentJSON(c))
}

func decodeArticlePatch(r *http.Request) (article.UpdatePatch, bool) {
	var body map[string]any
	if err := json.NewDecoder(io.LimitReader(r.Body, 4<<20)).Decode(&body); err != nil {
		return article.UpdatePatch{}, false
	}
	var patch article.UpdatePatch
	if v, ok := body["title"].(string); ok {
		patch.Title = &v
	}
	if v, ok := body["excerpt"].(string); ok {
		patch.Excerpt = &v
	}
	if tags, ok := body["tags"].([]any); ok {
		patch.HasTags = true
		for _, t := range tags {
			if s, ok := t.(string); ok {
				patch.Tags = append(patch.Tags, s)
			}
		}
	}
	if media, ok := body["media"].(map[string]any); ok {
		if v, ok := media["imageUrl"].(string); ok {
			patch.ImageURL = &v
		}
		if v, ok := media["videoUrl"].(string); ok {
			patch.VideoURL = &v
		}
	}
	if content, ok := body["content"]; ok {
		b, _ := json.Marshal(content)
		s := string(b)
		patch.ContentJSON = &s
	}
	if v, ok := body["readingTimeMinutes"].(float64); ok {
		n := int(v)
		patch.ReadingTimeMinutes = &n
	}
	return patch, true
}

func decodeDraftPatch(r *http.Request) (article.DraftPatch, bool) {
	var body map[string]any
	if err := json.NewDecoder(io.LimitReader(r.Body, 4<<20)).Decode(&body); err != nil {
		return article.DraftPatch{}, false
	}
	var patch article.DraftPatch
	if v, ok := body["title"].(string); ok {
		patch.Title = &v
	}
	if v, ok := body["excerpt"].(string); ok {
		patch.Excerpt = &v
	}
	if tags, ok := body["tags"].([]any); ok {
		patch.HasTags = true
		for _, t := range tags {
			if s, ok := t.(string); ok {
				patch.Tags = append(patch.Tags, s)
			}
		}
	}
	if media, ok := body["media"].(map[string]any); ok {
		if v, ok := media["imageUrl"].(string); ok {
			patch.ImageURL = &v
		}
		if v, ok := media["videoUrl"].(string); ok {
			patch.VideoURL = &v
		}
	}
	if content, ok := body["content"]; ok {
		b, _ := json.Marshal(content)
		s := string(b)
		patch.ContentJSON = &s
	}
	if v, ok := body["readingTimeMinutes"].(float64); ok {
		n := int(v)
		patch.ReadingTimeMinutes = &n
	}
	return patch, true
}