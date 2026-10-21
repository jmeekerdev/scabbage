package maps

import (
	"bytes"
	"compress/gzip"
	"errors"
	"path/filepath"
	"slices"
	"testing"

	"google.golang.org/protobuf/proto"

	"scabbage/internal/access"
	pb "scabbage/internal/pb/scabbage/v1"
	"scabbage/internal/store"
	"scabbage/internal/world"
	"scabbage/internal/zones"
)

func small() Map {
	m := Map{Name: "Castle", Width: 3, Depth: 2, Mode: access.ModeSurvival,
		Scripts: []Script{{Pos: world.Pos{X: 1, Y: 40, Z: 1}, Source: `say("hi")`, Owner: 7}},
		Start:   &Start{X: 1.5, Y: 41, Z: 0.5, Yaw: 1.25}}
	m.Blocks = make([]world.Block, m.Area(0, 0).Volume())
	for i := range m.Blocks {
		if i < 3*2*40 {
			m.Blocks[i] = world.Stone
		}
	}
	m.Blocks[m.Area(0, 0).Index(2, 40, 1)] = world.Chest
	return m
}

func TestEncodeDecodeRoundTrip(t *testing.T) {
	m := small()
	data, err := Encode(m)
	if err != nil {
		t.Fatal(err)
	}
	got, unknown, err := Decode(data)
	if err != nil || unknown != 0 {
		t.Fatalf("decode: %v, %d unknown", err, unknown)
	}
	if got.Name != m.Name || got.Width != 3 || got.Depth != 2 || got.Mode != m.Mode || *got.Start != *m.Start ||
		!slices.Equal(got.Blocks, m.Blocks) || !slices.Equal(got.Scripts, m.Scripts) {
		t.Fatalf("got %+v", got)
	}
}

func TestFilesKeepOnlyTheLayersInUse(t *testing.T) {
	m := small()
	data, _ := Encode(m)
	f, err := unzip(data)
	if err != nil || f.Height != 41-world.MinY {
		t.Fatalf("height %d (%v), want up to the chest at 40", f.GetHeight(), err)
	}
	full := rewrite(t, data, func(f *pb.MapFile) {
		f.Height, f.Blocks = world.MaxY-world.MinY, world.EncodeBlocks(m.Blocks)
	})
	if got, _, err := Decode(full); err != nil || !slices.Equal(got.Blocks, m.Blocks) {
		t.Fatalf("a file holding every layer: %v", err)
	}
}

func TestAreaPlacesTheMapAtACorner(t *testing.T) {
	a := small().Area(-10, 4)
	if a != (world.Area{X0: -10, Z0: 4, X1: -8, Z1: 5}) {
		t.Fatalf("area = %+v", a)
	}
}

// rewrite decodes a file, lets change edit it and encodes it again.
func rewrite(t *testing.T, data []byte, change func(*pb.MapFile)) []byte {
	t.Helper()
	r, err := gzip.NewReader(bytes.NewReader(data))
	if err != nil {
		t.Fatal(err)
	}
	var raw bytes.Buffer
	if _, err := raw.ReadFrom(r); err != nil {
		t.Fatal(err)
	}
	var f pb.MapFile
	if err := proto.Unmarshal(raw.Bytes(), &f); err != nil {
		t.Fatal(err)
	}
	change(&f)
	out, err := proto.Marshal(&f)
	if err != nil {
		t.Fatal(err)
	}
	var buf bytes.Buffer
	zw := gzip.NewWriter(&buf)
	_, _ = zw.Write(out)
	_ = zw.Close()
	return buf.Bytes()
}

func TestDecodeTurnsBlocksByName(t *testing.T) {
	data, _ := Encode(small())
	moved := rewrite(t, data, func(f *pb.MapFile) {
		for _, b := range f.Palette {
			if b.Name == "chest" {
				b.Name = "gold_block" // the chest's ID means gold here
			}
			if b.Name == "stone" {
				b.Name = "a block this server doesn't have"
			}
		}
	})
	got, unknown, err := Decode(moved)
	if err != nil {
		t.Fatal(err)
	}
	if b := got.Blocks[got.Area(0, 0).Index(2, 40, 1)]; b != world.GoldBlock {
		t.Fatalf("chest's spot holds %v", b)
	}
	if unknown != 3*2*40 || got.Blocks[0] != world.Air {
		t.Fatalf("unknown %d, first block %v", unknown, got.Blocks[0])
	}
}

func TestDecodeRejectsBadFiles(t *testing.T) {
	data, _ := Encode(small())
	bad := map[string][]byte{
		"not gzip":      []byte("hello"),
		"too tall":      rewrite(t, data, func(f *pb.MapFile) { f.Height = world.MaxY - world.MinY + 1 }),
		"no height":     rewrite(t, data, func(f *pb.MapFile) { f.Height = 0 }),
		"wrong height":  rewrite(t, data, func(f *pb.MapFile) { f.Height = 64 }),
		"too wide":      rewrite(t, data, func(f *pb.MapFile) { f.Width = MaxSide + 1 }),
		"no width":      rewrite(t, data, func(f *pb.MapFile) { f.Width = 0 }),
		"short blocks":  rewrite(t, data, func(f *pb.MapFile) { f.Width = 4 }),
		"script out":    rewrite(t, data, func(f *pb.MapFile) { f.Scripts[0].X = 3 }),
		"script length": rewrite(t, data, func(f *pb.MapFile) { f.Scripts[0].Source = string(make([]byte, 70000)) }),
		"bad mode":      rewrite(t, data, func(f *pb.MapFile) { f.Mode = "flying" }),
		"start out":     rewrite(t, data, func(f *pb.MapFile) { f.Start.X = 9 }),
	}
	for name, b := range bad {
		if _, _, err := Decode(b); !errors.Is(err, ErrBadFile) {
			t.Errorf("%s: %v", name, err)
		}
	}
}

func TestDecodeStopsAtHugeContents(t *testing.T) {
	var buf bytes.Buffer
	zw := gzip.NewWriter(&buf)
	_, _ = zw.Write(make([]byte, maxRawBytes+1))
	_ = zw.Close()
	if _, _, err := Decode(buf.Bytes()); !errors.Is(err, ErrBadFile) {
		t.Fatalf("err = %v", err)
	}
}

func openService(t *testing.T) (*Service, *zones.Service) {
	t.Helper()
	db, err := store.Open(filepath.Join(t.TempDir(), "m.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	z, err := zones.New(db, 0)
	if err != nil {
		t.Fatal(err)
	}
	return New(db), z
}

func TestServiceKeepsAMapPerZone(t *testing.T) {
	s, z := openService(t)
	if err := s.Put("Castle", small()); !errors.Is(err, zones.ErrNoZone) {
		t.Fatalf("saved without a zone: %v", err)
	}
	if err := z.Put("Castle", 0, 0, 2, 1); err != nil {
		t.Fatal(err)
	}
	if err := s.Put("castle", small()); err != nil {
		t.Fatal(err)
	}
	got, ok, err := s.Get("CASTLE")
	if err != nil || !ok || !slices.Equal(got.Blocks, small().Blocks) {
		t.Fatalf("get: %v %v", ok, err)
	}
	if names, _ := s.Names(); !slices.Equal(names, []string{"Castle"}) {
		t.Fatalf("names = %v", names)
	}
	if err := z.Remove("Castle"); err != nil {
		t.Fatal(err)
	}
	if _, ok, _ := s.Get("Castle"); ok {
		t.Fatal("map outlived its zone")
	}
	if err := s.Remove("Castle"); !errors.Is(err, ErrNoMap) {
		t.Fatalf("remove: %v", err)
	}
}

func TestServiceKeepsTheLandUnderAPlacedMap(t *testing.T) {
	s, z := openService(t)
	if err := z.Put("Castle", 0, 0, 2, 1); err != nil {
		t.Fatal(err)
	}
	if _, ok, _ := s.Under("Castle"); ok {
		t.Fatal("land found before any was kept")
	}
	land := small()
	land.Start = nil
	land.Blocks[0] = world.Brick
	if err := s.Put("Castle", small()); err != nil {
		t.Fatal(err)
	}
	if err := s.PutUnder("Castle", land); err != nil {
		t.Fatal(err)
	}
	if err := s.Put("Castle", small()); err != nil { // saving the map again keeps the land
		t.Fatal(err)
	}
	got, ok, err := s.Under("castle")
	if err != nil || !ok || got.Blocks[0] != world.Brick || len(got.Scripts) != 1 {
		t.Fatalf("under: %v %v", ok, err)
	}
	if err := s.PutUnder("Nowhere", land); !errors.Is(err, zones.ErrNoZone) {
		t.Fatalf("kept land without a zone: %v", err)
	}
}
