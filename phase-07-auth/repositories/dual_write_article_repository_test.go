package repositories

import (
	"context"
	"errors"
	"testing"

	"warehouse.local/core/entities"
	"warehouse.local/core/interfaces"
)

// recordingRepo is a test double that logs calls to a shared journal and can
// mint ids on Save (mimicking the legacy store, whose AUTO_INCREMENT assigns
// the aggregate id on insert).
type recordingRepo struct {
	name       string
	journal    *[]string
	mintID     string // if non-empty, assigned to a.ID when Save sees an empty id
	savedIDs   []string
	failOnSave error
}

var _ interfaces.ArticleRepository = (*recordingRepo)(nil)

func (r *recordingRepo) Save(ctx context.Context, a *entities.Article) error {
	*r.journal = append(*r.journal, r.name+".save")
	if r.failOnSave != nil {
		return r.failOnSave
	}
	if a.ID == "" && r.mintID != "" {
		a.ID = r.mintID
	}
	r.savedIDs = append(r.savedIDs, a.ID)
	return nil
}

func (r *recordingRepo) FindByID(ctx context.Context, id string) (*entities.Article, error) {
	*r.journal = append(*r.journal, r.name+".findByID")
	return nil, ErrArticleNotFound
}

func (r *recordingRepo) FindBySKU(ctx context.Context, skuCode string) (*entities.Article, error) {
	*r.journal = append(*r.journal, r.name+".findBySKU")
	return nil, ErrArticleNotFound
}

func (r *recordingRepo) List(ctx context.Context) ([]*entities.Article, error) {
	*r.journal = append(*r.journal, r.name+".list")
	return nil, nil
}

func (r *recordingRepo) Delete(ctx context.Context, id string) error {
	*r.journal = append(*r.journal, r.name+".delete")
	return nil
}

func newTestArticle(t *testing.T, id string) *entities.Article {
	t.Helper()
	sku, _ := entities.NewSKU("DW-001")
	price, _ := entities.NewMoney(990, "EUR")
	a, err := entities.NewArticle(id, *sku, "Dual write demo", "", *price)
	if err != nil {
		t.Fatalf("NewArticle: %v", err)
	}
	return a
}

func TestDualWrite_SaveWritesLegacyFirstAndPropagatesMintedID(t *testing.T) {
	var journal []string
	legacy := &recordingRepo{name: "legacy", journal: &journal, mintID: "42"}
	bc := &recordingRepo{name: "bc", journal: &journal}
	repo := NewDualWriteArticleRepository(legacy, bc, ReadFromLegacy)

	a := newTestArticle(t, "") // empty id: minted by the system of record
	if err := repo.Save(context.Background(), a); err != nil {
		t.Fatalf("Save: %v", err)
	}
	if len(journal) != 2 || journal[0] != "legacy.save" || journal[1] != "bc.save" {
		t.Errorf("expected legacy-first write order, got %v", journal)
	}
	if a.ID != "42" {
		t.Errorf("expected aggregate to carry the legacy-minted id 42, got %q", a.ID)
	}
	if len(bc.savedIDs) != 1 || bc.savedIDs[0] != "42" {
		t.Errorf("expected BC store to persist the SAME minted id 42, got %v", bc.savedIDs)
	}
}

func TestDualWrite_LegacyFailureStopsBeforeBC(t *testing.T) {
	var journal []string
	boom := errors.New("legacy down")
	legacy := &recordingRepo{name: "legacy", journal: &journal, failOnSave: boom}
	bc := &recordingRepo{name: "bc", journal: &journal}
	repo := NewDualWriteArticleRepository(legacy, bc, ReadFromLegacy)

	err := repo.Save(context.Background(), newTestArticle(t, "1"))
	if !errors.Is(err, boom) {
		t.Fatalf("expected legacy error to surface, got %v", err)
	}
	for _, entry := range journal {
		if entry == "bc.save" {
			t.Errorf("BC must not be written when legacy fails; journal=%v", journal)
		}
	}
}

func TestDualWrite_ReadsRouteToExactlyOneStore(t *testing.T) {
	cases := []struct {
		mode ReadMode
		want string
	}{
		{ReadFromLegacy, "legacy"},
		{ReadFromBC, "bc"},
	}
	for _, c := range cases {
		var journal []string
		legacy := &recordingRepo{name: "legacy", journal: &journal}
		bc := &recordingRepo{name: "bc", journal: &journal}
		repo := NewDualWriteArticleRepository(legacy, bc, c.mode)

		_, _ = repo.List(context.Background())
		_, _ = repo.FindByID(context.Background(), "1")
		_, _ = repo.FindBySKU(context.Background(), "DW-001")

		want := []string{c.want + ".list", c.want + ".findByID", c.want + ".findBySKU"}
		if len(journal) != len(want) {
			t.Fatalf("mode %v: expected reads on %s only, got %v", c.mode, c.want, journal)
		}
		for i := range want {
			if journal[i] != want[i] {
				t.Errorf("mode %v: journal[%d] = %q, want %q", c.mode, i, journal[i], want[i])
			}
		}
	}
}
