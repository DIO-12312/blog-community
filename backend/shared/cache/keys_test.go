package cache

import (
	"testing"
)

func TestArticleKey(t *testing.T) {
	tests := []struct {
		name      string
		articleID string
		want      string
	}{
		{"normal id", "abc123", "article:abc123"},
		{"uuid format", "550e8400-e29b-41d4-a716-446655440000", "article:550e8400-e29b-41d4-a716-446655440000"},
		{"numeric id", "12345", "article:12345"},
		{"empty id", "", "article:"},
		{"special chars", "test_article-01", "article:test_article-01"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ArticleKey(tt.articleID)
			if got != tt.want {
				t.Errorf("ArticleKey(%q) = %q, want %q", tt.articleID, got, tt.want)
			}
		})
	}
}

func TestArticleListKey(t *testing.T) {
	tests := []struct {
		name     string
		category string
		page     int
		size     int
		want     string
	}{
		{"with category", "tech", 1, 10, "articles:tech:1:10"},
		{"empty category (all)", "", 1, 20, "articles::1:20"},
		{"page 0", "life", 0, 5, "articles:life:0:5"},
		{"large page", "ai", 999, 50, "articles:ai:999:50"},
		{"chinese category", "科技", 2, 15, "articles:科技:2:15"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ArticleListKey(tt.category, tt.page, tt.size)
			if got != tt.want {
				t.Errorf("ArticleListKey(%q, %d, %d) = %q, want %q", tt.category, tt.page, tt.size, got, tt.want)
			}
		})
	}
}

func TestViewCountKey(t *testing.T) {
	tests := []struct {
		name      string
		articleID string
		want      string
	}{
		{"normal id", "abc123", "view_count:abc123"},
		{"uuid", "550e8400-e29b-41d4-a716-446655440000", "view_count:550e8400-e29b-41d4-a716-446655440000"},
		{"empty", "", "view_count:"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ViewCountKey(tt.articleID)
			if got != tt.want {
				t.Errorf("ViewCountKey(%q) = %q, want %q", tt.articleID, got, tt.want)
			}
		})
	}
}

func TestUserKey(t *testing.T) {
	tests := []struct {
		name   string
		userID string
		want   string
	}{
		{"normal id", "user123", "user:user123"},
		{"uuid", "550e8400-e29b-41d4-a716-446655440000", "user:550e8400-e29b-41d4-a716-446655440000"},
		{"empty", "", "user:"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := UserKey(tt.userID)
			if got != tt.want {
				t.Errorf("UserKey(%q) = %q, want %q", tt.userID, got, tt.want)
			}
		})
	}
}

func TestCommentListKey(t *testing.T) {
	tests := []struct {
		name      string
		articleID string
		page      int
		size      int
		want      string
	}{
		{"normal", "article_001", 1, 10, "comments:article_001:1:10"},
		{"uuid article", "550e8400-e29b-41d4-a716-446655440000", 2, 20, "comments:550e8400-e29b-41d4-a716-446655440000:2:20"},
		{"zero values", "", 0, 0, "comments::0:0"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := CommentListKey(tt.articleID, tt.page, tt.size)
			if got != tt.want {
				t.Errorf("CommentListKey(%q, %d, %d) = %q, want %q", tt.articleID, tt.page, tt.size, got, tt.want)
			}
		})
	}
}

func TestNullValue(t *testing.T) {
	if NullValue != "__NULL__" {
		t.Errorf("NullValue = %q, want %q", NullValue, "__NULL__")
	}
}

func TestExpirationConstants(t *testing.T) {
	tests := []struct {
		name  string
		value int
		want  int
	}{
		{"ArticleExpiration", ArticleExpiration, 86400},
		{"ArticleListExpiration", ArticleListExpiration, 3600},
		{"UserExpiration", UserExpiration, 43200},
		{"CommentListExpiration", CommentListExpiration, 1800},
		{"EmptyValueExpiration", EmptyValueExpiration, 300},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.value != tt.want {
				t.Errorf("%s = %d, want %d", tt.name, tt.value, tt.want)
			}
		})
	}
}
