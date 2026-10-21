package world

import "bytes"

// AddDungeons puts dungeons into a version 2 world's saved land wherever every chunk they touch
// still matches version 1's untouched land, by forgetting those chunks so they generate again.
// Call it before players join; it returns how many dungeons it added.
func (w *World) AddDungeons() (int, error) {
	g, ok := w.gen.(*GenV2)
	if !ok || w.meta.Radius == Unbounded {
		return 0, nil
	}
	w.mu.Lock()
	defer w.mu.Unlock()
	added := 0
	last := (w.meta.Radius + regionSize - 1) / regionSize
	for rx := -last; rx < last; rx++ {
		for rz := -last; rz < last; rz++ {
			d, ok := g.dungeonIn(rx, rz)
			if !ok {
				continue
			}
			stale, err := w.untouched(g, d.chunks())
			if err != nil {
				return added, err
			}
			if len(stale) > 0 {
				if err := w.store.deleteChunks(w.meta.ID, stale); err != nil {
					return added, err
				}
				added++
			}
		}
	}
	return added, nil
}

// untouched lists the saved chunks among ps when all of them hold only version 1 land, else none.
func (w *World) untouched(g *GenV2, ps []ChunkPos) ([]ChunkPos, error) {
	var saved []ChunkPos
	for _, p := range ps {
		if _, loaded := w.chunks[p]; loaded || !w.Inside(p) {
			return nil, nil
		}
		data, err := w.store.loadChunk(w.meta.ID, p)
		if err != nil {
			return nil, err
		}
		if data == nil {
			continue
		}
		if !bytes.Equal(data, Encode(g.GenV1.Generate(p))) {
			return nil, nil
		}
		saved = append(saved, p)
	}
	return saved, nil
}
