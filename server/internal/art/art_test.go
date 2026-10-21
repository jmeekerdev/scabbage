package art

import (
	"bytes"
	"database/sql"
	"errors"
	"path/filepath"
	"testing"

	"scabbage/internal/store"
	"scabbage/internal/world"
)

type fixture struct {
	svc     *Service
	db      *sql.DB
	changes int
}

func newFixture(t *testing.T) *fixture {
	t.Helper()
	db, err := store.Open(filepath.Join(t.TempDir(), "art.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	f := &fixture{db: db}
	f.svc = New(db, Options{
		OnPendingChange: func() { f.changes++ },
		Reserved:        func(name string) bool { return name == "stone" },
	})
	return f
}

func (f *fixture) player(t *testing.T, name string) int64 {
	t.Helper()
	res, err := f.db.Exec(`INSERT INTO players (name, color, admin, created_ms) VALUES (?, 255, 0, 0)`, name)
	if err != nil {
		t.Fatal(err)
	}
	id, _ := res.LastInsertId()
	return id
}

// pixels is size opaque pixels of one gray.
func pixels(size int, gray byte) []byte { return bytes.Repeat([]byte{gray, gray, gray, 255}, size) }

var (
	block = pixels(BlockSize*BlockSize, 200)
	skin  = pixels(SkinWidth*SkinHeight, 200)
)

func TestPlayerArtWaitsForApproval(t *testing.T) {
	f := newFixture(t)
	mia := f.player(t, "Mia")
	p, err := f.svc.Submit(mia, false, Block, " Flower tile ", block)
	if err != nil {
		t.Fatal(err)
	}
	if p.Approved || p.Name != "Flower tile" || p.Author != "Mia" || f.changes != 1 {
		t.Fatalf("piece = %+v, changes = %d", p, f.changes)
	}
	pending, err := f.svc.Pending()
	if err != nil || len(pending) != 1 || pending[0].ID != p.ID || !bytes.Equal(pending[0].Pixels, block) {
		t.Fatalf("pending = %+v, %v", pending, err)
	}
	if blocks, _ := f.svc.Blocks(); len(blocks) != 0 {
		t.Fatalf("unapproved block listed: %+v", blocks)
	}
}

func TestApprovedBlocksGetStableIDs(t *testing.T) {
	f := newFixture(t)
	mia := f.player(t, "Mia")
	first, _ := f.svc.Submit(mia, false, Block, "one", block)
	second, _ := f.svc.Submit(mia, false, Block, "two", block)
	if _, err := f.svc.Decide(second.ID, true); err != nil {
		t.Fatal(err)
	}
	got, err := f.svc.Decide(first.ID, true)
	if err != nil {
		t.Fatal(err)
	}
	if got.BlockID != world.FirstCustom+1 || !got.Approved {
		t.Fatalf("approved = %+v", got)
	}
	blocks, _ := f.svc.Blocks()
	if len(blocks) != 2 || blocks[0].Name != "two" || blocks[0].BlockID != world.FirstCustom {
		t.Fatalf("blocks = %+v", blocks)
	}
}

func TestDeniedArtIsDropped(t *testing.T) {
	f := newFixture(t)
	p, _ := f.svc.Submit(f.player(t, "Mia"), false, Skin, "", skin)
	if _, err := f.svc.Decide(p.ID, false); err != nil {
		t.Fatal(err)
	}
	if pending, _ := f.svc.Pending(); len(pending) != 0 {
		t.Fatalf("pending = %+v", pending)
	}
	if _, err := f.svc.Decide(p.ID, true); !errors.Is(err, ErrNotFound) {
		t.Fatalf("second decision err = %v", err)
	}
}

func TestAdminArtIsApprovedAtOnce(t *testing.T) {
	f := newFixture(t)
	p, err := f.svc.Submit(f.player(t, "Dad"), true, Block, "tile", block)
	if err != nil || !p.Approved || p.BlockID != world.FirstCustom {
		t.Fatalf("piece = %+v, %v", p, err)
	}
	if f.changes != 0 {
		t.Fatal("pending list changed for admin art")
	}
}

func TestNewSkinReplacesOld(t *testing.T) {
	f := newFixture(t)
	mia := f.player(t, "Mia")
	if s, _ := f.svc.Skin(mia); s != nil {
		t.Fatal("skin before any was made")
	}
	_, _ = f.svc.Submit(mia, true, Skin, "", skin)
	newer := pixels(SkinWidth*SkinHeight, 255)
	_, _ = f.svc.Submit(mia, true, Skin, "", newer)
	if s, _ := f.svc.Skin(mia); !bytes.Equal(s, newer) {
		t.Fatal("skin is not the newest")
	}
	var n int
	_ = f.db.QueryRow(`SELECT count(*) FROM art`).Scan(&n)
	if n != 1 {
		t.Fatalf("%d rows kept, want 1", n)
	}
}

func TestAnyoneCanResetTheirSkinAtOnce(t *testing.T) {
	f := newFixture(t)
	mia := f.player(t, "Mia")
	_, _ = f.svc.Submit(mia, true, Skin, "", skin)
	p, err := f.svc.Submit(mia, false, Skin, "", nil)
	if err != nil || !p.Approved || p.Pixels != nil {
		t.Fatalf("reset = %+v, %v", p, err)
	}
	if s, _ := f.svc.Skin(mia); s != nil {
		t.Fatal("skin still set after reset")
	}
	if f.changes != 0 {
		t.Fatal("a reset changed the waiting list")
	}
}

func TestTexturesAreAdminOnlyAndResettable(t *testing.T) {
	f := newFixture(t)
	mia := f.player(t, "Mia")
	if _, err := f.svc.Submit(mia, false, Texture, "grass_top", block); !errors.Is(err, ErrNotAdmin) {
		t.Fatalf("player repaint err = %v", err)
	}
	if _, err := f.svc.Submit(mia, true, Texture, "grass_top", block); err != nil {
		t.Fatal(err)
	}
	if tex, _ := f.svc.Textures(); len(tex) != 1 || tex[0].Name != "grass_top" {
		t.Fatalf("textures = %+v", tex)
	}
	if _, err := f.svc.Submit(mia, true, Texture, "grass_top", nil); err != nil {
		t.Fatal(err)
	}
	if tex, _ := f.svc.Textures(); len(tex) != 0 {
		t.Fatalf("textures after reset = %+v", tex)
	}
}

func TestSubmitValidates(t *testing.T) {
	f := newFixture(t)
	mia := f.player(t, "Mia")
	_, _ = f.svc.Submit(mia, false, Block, "Tile", block)
	cases := []struct {
		kind   Kind
		name   string
		pixels []byte
		want   error
	}{
		{Block, "x", skin, ErrBadSize},
		{Skin, "", block, ErrBadSize},
		{Block, "", block, ErrBadName},
		{Block, "<script>", block, ErrBadName},
		{Block, "Stone", block, ErrNameTaken},
		{Block, "tile", block, ErrNameTaken},
		{Texture, "Grass Top", block, ErrBadName},
		{Kind(9), "", block, ErrBadKind},
	}
	for _, c := range cases {
		if _, err := f.svc.Submit(mia, true, c.kind, c.name, c.pixels); !errors.Is(err, c.want) {
			t.Errorf("Submit(%v, %q) err = %v, want %v", c.kind, c.name, err, c.want)
		}
	}
}

func TestPlayersCanOnlyHaveAFewWaiting(t *testing.T) {
	f := newFixture(t)
	mia := f.player(t, "Mia")
	for range maxPendingEach {
		if _, err := f.svc.Submit(mia, false, Skin, "", skin); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := f.svc.Submit(mia, false, Skin, "", skin); !errors.Is(err, ErrTooMany) {
		t.Fatalf("err = %v", err)
	}
}

func TestBlockLimit(t *testing.T) {
	f := newFixture(t)
	dad := f.player(t, "Dad")
	for i := range MaxBlocks {
		if _, err := f.svc.Submit(dad, true, Block, string(rune('A'+i/26))+string(rune('a'+i%26)), block); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := f.svc.Submit(dad, true, Block, "one more", block); !errors.Is(err, ErrFull) {
		t.Fatalf("err = %v", err)
	}
}

func TestPixelsAreOpaqueOrClear(t *testing.T) {
	f := newFixture(t)
	px := bytes.Repeat([]byte{90}, BlockSize*BlockSize*4)
	px[3], px[7] = 200, 255 // first pixel mostly opaque, second solid; the rest faint
	p, err := f.svc.Submit(f.player(t, "Dad"), true, Block, "tile", px)
	if err != nil {
		t.Fatal(err)
	}
	if p.Pixels[3] != 255 || p.Pixels[7] != 255 || !bytes.Equal(p.Pixels[8:12], []byte{0, 0, 0, 0}) {
		t.Fatalf("pixels = %v", p.Pixels[:12])
	}
	if !p.Transparent() {
		t.Fatal("block with clear pixels is not transparent")
	}
}

func TestWipeForgetsAllArt(t *testing.T) {
	f := newFixture(t)
	mia := f.player(t, "Mia")
	_, _ = f.svc.Submit(mia, true, Block, "one", block)
	_, _ = f.svc.Submit(mia, false, Block, "two", block)
	if err := f.svc.Wipe(); err != nil {
		t.Fatal(err)
	}
	blocks, _ := f.svc.Blocks()
	pending, _ := f.svc.Pending()
	if len(blocks) != 0 || len(pending) != 0 {
		t.Fatalf("blocks %v, pending %v", blocks, pending)
	}
	if f.changes == 0 {
		t.Fatal("admins weren't told the pending art went")
	}
}
