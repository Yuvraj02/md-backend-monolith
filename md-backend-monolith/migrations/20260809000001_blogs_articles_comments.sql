-- Replace the tiny boilerplate schema with blogs + articles + comments.
-- Schema only — no seed data.

DROP TABLE IF EXISTS "articles";

CREATE TABLE IF NOT EXISTS "blogs" (
  "id" text PRIMARY KEY,
  "name" text NOT NULL,
  "slug" text NOT NULL UNIQUE,
  "description" text NOT NULL DEFAULT '',
  "cover_image" text NOT NULL DEFAULT '',
  "owner_id" text NOT NULL,
  "upvotes" bigint NOT NULL DEFAULT 0,
  "total_views" bigint NOT NULL DEFAULT 0,
  "article_count" integer NOT NULL DEFAULT 0,
  "total_reading_time_minutes" integer NOT NULL DEFAULT 0,
  "created_at" timestamptz NOT NULL,
  "updated_at" timestamptz NOT NULL
);

CREATE TABLE IF NOT EXISTS "articles" (
  "id" text PRIMARY KEY,
  "blog_id" text NOT NULL REFERENCES "blogs" ("id") ON DELETE CASCADE,
  "title" text NOT NULL DEFAULT '',
  "excerpt" text NOT NULL DEFAULT '',
  "image_url" text NOT NULL DEFAULT '',
  "video_url" text NOT NULL DEFAULT '',
  "tags_json" text NOT NULL DEFAULT '[]',
  "content_json" text NOT NULL DEFAULT '[]',
  "is_draft" boolean NOT NULL DEFAULT true,
  "upvotes" bigint NOT NULL DEFAULT 0,
  "views" bigint NOT NULL DEFAULT 0,
  "views_this_week" bigint NOT NULL DEFAULT 0,
  "reads" bigint NOT NULL DEFAULT 0,
  "reading_time_minutes" integer NOT NULL DEFAULT 1,
  "published_at" timestamptz NULL,
  "last_saved" timestamptz NULL,
  "created_at" timestamptz NOT NULL,
  "updated_at" timestamptz NOT NULL
);

CREATE INDEX IF NOT EXISTS "idx_articles_blog_id" ON "articles" ("blog_id");
CREATE INDEX IF NOT EXISTS "idx_articles_published" ON "articles" ("is_draft", "published_at");

CREATE TABLE IF NOT EXISTS "comments" (
  "id" text PRIMARY KEY,
  "article_id" text NOT NULL REFERENCES "articles" ("id") ON DELETE CASCADE,
  "name" text NOT NULL,
  "email" text NOT NULL,
  "website" text NOT NULL DEFAULT '',
  "body" text NOT NULL,
  "created_at" timestamptz NOT NULL
);

CREATE INDEX IF NOT EXISTS "idx_comments_article_id" ON "comments" ("article_id");
