package usecases

// MockArticleRepository is an alias for the in-memory test fake (see
// in_memory_article_repository.go). Pre-existing tests refer to this name; the
// alias keeps them readable while delegating to the canonical implementation.
type MockArticleRepository = InMemoryArticleRepository

// NewMockArticleRepository constructs a MockArticleRepository (alias for
// NewInMemoryArticleRepository).
func NewMockArticleRepository() *MockArticleRepository {
	return NewInMemoryArticleRepository()
}
