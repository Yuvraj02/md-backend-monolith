package httpserver

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/marketing-digest/md-backend-monolith/internal/app/article"
	"github.com/marketing-digest/md-backend-monolith/internal/app/blog"
	"github.com/marketing-digest/md-backend-monolith/internal/middleware"
)

// Server is the HTTP API: handlers translate HTTP ↔ domain services.
type Server struct {
	http         *http.Server
	blogs        *blog.Service
	articles     *article.Service
	log          *slog.Logger
	ownerSecret  string
}

func New(addr string, blogs *blog.Service, articles *article.Service, log *slog.Logger, corsOrigins, ownerSecret string) *Server {
	mux := http.NewServeMux()
	s := &Server{blogs: blogs, articles: articles, log: log, ownerSecret: ownerSecret}

	mux.HandleFunc("GET /healthz", s.handleLiveness)
	mux.HandleFunc("GET /readyz", s.handleReadiness)
	mux.HandleFunc("GET /v1/blog/ping", s.handleBlogPing)

	mux.HandleFunc("GET /v1/blogs", s.handleListBlogs)
	mux.HandleFunc("GET /v1/blogs/{blogID}", s.handleGetBlog)
	mux.HandleFunc("POST /v1/blogs/{blogID}/upvotes", s.handleUpvoteBlog)

	mux.Handle("POST /v1/blogs", s.requireOwner(http.HandlerFunc(s.handleCreateBlog)))
	mux.Handle("PATCH /v1/blogs/{blogID}", s.requireOwner(http.HandlerFunc(s.handleUpdateBlog)))

	mux.HandleFunc("GET /v1/blogs/{blogID}/articles", s.handleListArticles)
	mux.HandleFunc("GET /v1/blogs/{blogID}/articles/{articleID}", s.handleGetArticle)
	mux.Handle("POST /v1/blogs/{blogID}/articles", s.requireOwner(http.HandlerFunc(s.handleCreateArticle)))
	mux.Handle("PATCH /v1/blogs/{blogID}/articles/{articleID}", s.requireOwner(http.HandlerFunc(s.handleUpdateArticle)))
	mux.Handle("DELETE /v1/blogs/{blogID}/articles/{articleID}", s.requireOwner(http.HandlerFunc(s.handleDeleteArticle)))
	mux.HandleFunc("POST /v1/blogs/{blogID}/articles/{articleID}/upvotes", s.handleUpvoteArticle)
	mux.Handle("POST /v1/blogs/{blogID}/articles/{articleID}/publish", s.requireOwner(http.HandlerFunc(s.handlePublishArticle)))
	mux.Handle("POST /v1/blogs/{blogID}/articles/{articleID}/draft", s.requireOwner(http.HandlerFunc(s.handleEnsureDraft)))
	mux.Handle("GET /v1/blogs/{blogID}/articles/{articleID}/draft", s.requireOwner(http.HandlerFunc(s.handleGetDraft)))
	mux.Handle("PATCH /v1/blogs/{blogID}/articles/{articleID}/draft", s.requireOwner(http.HandlerFunc(s.handleUpdateDraft)))
	mux.Handle("POST /v1/blogs/{blogID}/articles/{articleID}/draft/publish", s.requireOwner(http.HandlerFunc(s.handlePublishDraft)))

	mux.HandleFunc("GET /v1/articles", s.handleListPublished)
	mux.HandleFunc("GET /v1/articles/trending", s.handleListTrending)
	mux.HandleFunc("GET /v1/articles/{articleID}", s.handleGetPublishedArticle)
	mux.HandleFunc("GET /v1/articles/{articleID}/related", s.handleListRelated)
	mux.HandleFunc("POST /v1/articles/{articleID}/views", s.handleRecordView)
	mux.HandleFunc("POST /v1/articles/{articleID}/reads", s.handleRecordRead)
	mux.HandleFunc("GET /v1/articles/{articleID}/comments", s.handleListComments)
	mux.HandleFunc("POST /v1/articles/{articleID}/comments", s.handleCreateComment)

	handler := middleware.Chain(
		mux,
		middleware.WithRecovery(log),
		middleware.WithRequestID,
		middleware.WithLogging(log),
		middleware.WithCORS(corsOrigins),
	)

	s.http = &http.Server{
		Addr:              addr,
		Handler:           handler,
		ReadHeaderTimeout: 5 * time.Second,
	}
	return s
}

func (s *Server) requireOwner(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if s.ownerSecret == "" || r.Header.Get("X-Owner-Secret") != s.ownerSecret {
			writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "owner secret required"})
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (s *Server) ListenAndServe() error {
	s.log.Info("http listening", "addr", s.http.Addr)
	err := s.http.ListenAndServe()
	if errors.Is(err, http.ErrServerClosed) {
		return nil
	}
	return err
}

func (s *Server) Shutdown(ctx context.Context) error {
	return s.http.Shutdown(ctx)
}

func (s *Server) Handler() http.Handler {
	return s.http.Handler
}

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}