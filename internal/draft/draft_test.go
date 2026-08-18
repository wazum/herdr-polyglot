package draft_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/wazum/herdr-polyglot/internal/draft"
)

func TestADraftComesBackForThePaneItWasWrittenFor(t *testing.T) {
	t.Parallel()
	store := draft.NewStore(t.TempDir())

	if err := store.For("w1:p3").Save("Bitte behebe den Test"); err != nil {
		t.Fatalf("Save returned unexpected error: %v", err)
	}

	if kept, _ := store.For("w1:p3").Load(); kept != "Bitte behebe den Test" {
		t.Errorf("Load returned %q, want the saved draft", kept)
	}
	if other, _ := store.For("w1:p9").Load(); other != "" {
		t.Errorf("another pane sees %q, want its own empty draft", other)
	}
}

func TestASentDraftIsForgotten(t *testing.T) {
	t.Parallel()
	store := draft.NewStore(t.TempDir())
	slot := store.For("w1:p3")

	if err := slot.Save("Bitte behebe den Test"); err != nil {
		t.Fatalf("Save returned unexpected error: %v", err)
	}
	if err := slot.Clear(); err != nil {
		t.Fatalf("Clear returned unexpected error: %v", err)
	}

	if kept, _ := slot.Load(); kept != "" {
		t.Errorf("Load returned %q after clearing, want nothing", kept)
	}
}

func TestSavingNothingLeavesNoFileBehind(t *testing.T) {
	t.Parallel()
	directory := t.TempDir()
	slot := draft.NewStore(directory).For("w1:p3")

	if err := slot.Save("Bitte behebe den Test"); err != nil {
		t.Fatalf("Save returned unexpected error: %v", err)
	}
	if err := slot.Save("   \n  "); err != nil {
		t.Fatalf("Save returned unexpected error: %v", err)
	}

	entries, err := os.ReadDir(directory)
	if err != nil {
		t.Fatalf("reading the store: %v", err)
	}
	if len(entries) != 0 {
		t.Errorf("the store holds %d files, want a blank draft to remove its own", len(entries))
	}
}

func TestADraftIsReadableOnlyByItsAuthor(t *testing.T) {
	t.Parallel()
	directory := t.TempDir()

	if err := draft.NewStore(directory).For("w1:p3").Save("Bitte behebe den Test"); err != nil {
		t.Fatalf("Save returned unexpected error: %v", err)
	}

	entries, err := os.ReadDir(directory)
	if err != nil || len(entries) != 1 {
		t.Fatalf("reading the store: %v (%d entries)", err, len(entries))
	}
	info, err := os.Stat(filepath.Join(directory, entries[0].Name()))
	if err != nil {
		t.Fatalf("stat: %v", err)
	}
	// A draft is unfinished thinking about the user's own code.
	if mode := info.Mode().Perm(); mode&0o077 != 0 {
		t.Errorf("the draft file is %v, want it private", mode)
	}
}

func TestAPaneIdNeverEscapesTheStoreDirectory(t *testing.T) {
	t.Parallel()
	directory := t.TempDir()
	store := draft.NewStore(directory)

	if err := store.For("../../escaped").Save("Bitte behebe den Test"); err != nil {
		t.Fatalf("Save returned unexpected error: %v", err)
	}

	entries, err := os.ReadDir(directory)
	if err != nil {
		t.Fatalf("reading the store: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("the store holds %d files, want the draft inside it", len(entries))
	}
	if kept, _ := store.For("../../escaped").Load(); kept != "Bitte behebe den Test" {
		t.Errorf("Load returned %q, want the draft back", kept)
	}
}

func TestWithoutAStoreDirectoryNothingIsKeptAndNothingFails(t *testing.T) {
	t.Parallel()
	slot := draft.NewStore("").For("w1:p3")

	if err := slot.Save("Bitte behebe den Test"); err != nil {
		t.Errorf("Save returned %v, want a missing store to be no error", err)
	}
	if kept, _ := slot.Load(); kept != "" {
		t.Errorf("Load returned %q, want nothing", kept)
	}
	if err := slot.Clear(); err != nil {
		t.Errorf("Clear returned %v, want a missing store to be no error", err)
	}
}

func TestAnExistingFileWithLooseAccessIsTightenedOnSave(t *testing.T) {
	t.Parallel()
	directory := t.TempDir()
	slot := draft.NewStore(directory).For("w1:p3")

	if err := slot.Save("erste Fassung"); err != nil {
		t.Fatalf("Save returned unexpected error: %v", err)
	}
	entries, err := os.ReadDir(directory)
	if err != nil || len(entries) != 1 {
		t.Fatalf("reading the store: %v", err)
	}
	path := filepath.Join(directory, entries[0].Name())
	if err := os.Chmod(path, 0o644); err != nil {
		t.Fatalf("loosening the file: %v", err)
	}

	if err := slot.Save("zweite Fassung"); err != nil {
		t.Fatalf("Save returned unexpected error: %v", err)
	}

	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat: %v", err)
	}
	if mode := info.Mode().Perm(); mode&0o077 != 0 {
		t.Errorf("the draft file is %v, want saving to put it back to private", mode)
	}
}

func TestASymbolicLinkIsNotFollowedOutOfTheStore(t *testing.T) {
	t.Parallel()
	directory := t.TempDir()
	elsewhere := filepath.Join(t.TempDir(), "elsewhere")
	if err := os.WriteFile(elsewhere, []byte("do not touch"), 0o600); err != nil {
		t.Fatalf("writing the outside file: %v", err)
	}

	store := draft.NewStore(directory)
	slot := store.For("w1:p3")
	if err := slot.Save("erste Fassung"); err != nil {
		t.Fatalf("Save returned unexpected error: %v", err)
	}
	entries, _ := os.ReadDir(directory)
	path := filepath.Join(directory, entries[0].Name())
	if err := os.Remove(path); err != nil {
		t.Fatalf("removing the draft: %v", err)
	}
	if err := os.Symlink(elsewhere, path); err != nil {
		t.Fatalf("planting the link: %v", err)
	}

	_ = slot.Save("zweite Fassung")

	outside, err := os.ReadFile(elsewhere)
	if err != nil {
		t.Fatalf("reading the outside file: %v", err)
	}
	if string(outside) != "do not touch" {
		t.Errorf("the file outside the store now holds %q", outside)
	}
}

// A draft that is there but cannot be read is not the same as no draft: saying so
// is the only warning before the next save writes over it.
func TestADraftThatCannotBeReadIsNotReportedAsAbsent(t *testing.T) {
	t.Parallel()
	directory := t.TempDir()
	store := draft.NewStore(directory)
	slot := store.For("w1:p1")

	if err := slot.Save("Bitte behebe den Test"); err != nil {
		t.Fatalf("Save returned unexpected error: %v", err)
	}

	kept, err := os.ReadDir(directory)
	if err != nil || len(kept) != 1 {
		t.Fatalf("the store holds %v (%v), want the one draft", kept, err)
	}
	unreadable := filepath.Join(directory, kept[0].Name())
	if err := os.Chmod(unreadable, 0o000); err != nil {
		t.Fatalf("making the draft unreadable: %v", err)
	}
	t.Cleanup(func() { _ = os.Chmod(unreadable, 0o600) })

	text, err := slot.Load()
	if err == nil {
		t.Error("Load returned no error for a draft it could not read")
	}
	if text != "" {
		t.Errorf("Load returned %q, want nothing alongside the error", text)
	}
}

func TestAMissingDraftIsSimplyEmpty(t *testing.T) {
	t.Parallel()

	text, err := draft.NewStore(t.TempDir()).For("w1:p1").Load()
	if err != nil {
		t.Errorf("Load returned %v for a pane with no draft, want no error", err)
	}
	if text != "" {
		t.Errorf("Load returned %q, want nothing", text)
	}
}
