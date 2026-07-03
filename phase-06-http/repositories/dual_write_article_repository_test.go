package repositories

import (
	"context"
	"errors"
	"testing"
)

// These tests are the acceptance criteria for Phase 04 Task 2. They run against
// in-memory fakes (fast, no MySQL) and are RED until you implement the decorator
// in dual_write_article_repository.go. Make them green.

func TestDualWriteArticleRepository_Save_writesToBoth(t *testing.T) {
	ctx := context.Background()
	legacy := NewInMemoryArticleRepository()
	bc := NewInMemoryArticleRepository()
	dw := NewDualWriteArticleRepository(legacy, bc, ReadFromLegacy)

	a := mustArticle(t, "id-1", "ABC-001", 1000)
	if err := dw.Save(ctx, a); err != nil {
		t.Fatalf("Save: %v", err)
	}
	if legacy.Count() != 1 {
		t.Error("expected article in legacy store")
	}
	if bc.Count() != 1 {
		t.Error("expected article in bc store")
	}
}

func TestDualWriteArticleRepository_Save_legacyFails_returnsErrorAndDoesNotWriteBC(t *testing.T) {
	ctx := context.Background()
	legacy := NewInMemoryArticleRepository()
	legacy.FailOnSave = errors.New("legacy down")
	bc := NewInMemoryArticleRepository()
	dw := NewDualWriteArticleRepository(legacy, bc, ReadFromLegacy)

	a := mustArticle(t, "id-1", "ABC-001", 1000)
	if err := dw.Save(ctx, a); err == nil {
		t.Fatal("expected error when legacy fails")
	}
	if bc.Count() != 0 {
		t.Error("expected BC to be untouched when legacy fails")
	}
}

func TestDualWriteArticleRepository_Save_bcFails_returnsError(t *testing.T) {
	ctx := context.Background()
	legacy := NewInMemoryArticleRepository()
	bc := NewInMemoryArticleRepository()
	bc.FailOnSave = errors.New("bc down")
	dw := NewDualWriteArticleRepository(legacy, bc, ReadFromLegacy)

	a := mustArticle(t, "id-1", "ABC-001", 1000)
	if err := dw.Save(ctx, a); err == nil {
		t.Fatal("expected error when bc fails")
	}
	// Legacy keeps the article: there is no rollback policy in this exercise.
	if legacy.Count() != 1 {
		t.Error("expected legacy to keep the article when BC fails")
	}
}

func TestDualWriteArticleRepository_FindByID_routesByMode(t *testing.T) {
	ctx := context.Background()
	legacy := NewInMemoryArticleRepository()
	bc := NewInMemoryArticleRepository()

	// Seed legacy and BC with different articles so we can tell which store answered.
	legacyOnly := mustArticle(t, "id-legacy", "AAA-001", 100)
	bcOnly := mustArticle(t, "id-bc", "BBB-001", 200)
	_ = legacy.Save(ctx, legacyOnly)
	_ = bc.Save(ctx, bcOnly)

	dwLegacy := NewDualWriteArticleRepository(legacy, bc, ReadFromLegacy)
	if got, _ := dwLegacy.FindByID(ctx, "id-legacy"); got == nil || got.ID != "id-legacy" {
		t.Errorf("expected ReadFromLegacy mode to find the legacy article")
	}

	dwBC := NewDualWriteArticleRepository(legacy, bc, ReadFromBC)
	if got, _ := dwBC.FindByID(ctx, "id-bc"); got == nil || got.ID != "id-bc" {
		t.Errorf("expected ReadFromBC mode to find the bc article")
	}
}

func TestDualWriteArticleRepository_Delete_writesToBoth(t *testing.T) {
	ctx := context.Background()
	legacy := NewInMemoryArticleRepository()
	bc := NewInMemoryArticleRepository()
	dw := NewDualWriteArticleRepository(legacy, bc, ReadFromLegacy)

	a := mustArticle(t, "id-1", "ABC-001", 1000)
	if err := dw.Save(ctx, a); err != nil {
		t.Fatalf("Save (setup): %v", err)
	}
	if err := dw.Delete(ctx, "id-1"); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if legacy.Count() != 0 || bc.Count() != 0 {
		t.Errorf("expected both stores empty after Delete; legacy=%d bc=%d", legacy.Count(), bc.Count())
	}
}
