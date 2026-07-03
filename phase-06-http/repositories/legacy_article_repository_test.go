package repositories

import "testing"

// TestLegacyMySQLArticleRepository_compilesAgainstInterface is a starter smoke
// check. The compile-time assertion lives in legacy_article_repository.go; this
// test keeps the participant starting point green before Step 8.
func TestLegacyMySQLArticleRepository_compilesAgainstInterface(t *testing.T) {
	r := NewLegacyMySQLArticleRepository(nil)
	if r == nil {
		t.Fatal("NewLegacyMySQLArticleRepository returned nil")
	}
}
