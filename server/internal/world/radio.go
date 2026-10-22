package world

// radios are the loaded wireless blocks.
type radios struct {
	transmitters map[Pos]struct{}
	receivers    map[Pos]struct{}
}

func newRadios() radios {
	return radios{transmitters: map[Pos]struct{}{}, receivers: map[Pos]struct{}{}}
}

// note records b arriving at p, reporting whether a transmitter came, went or changed.
func (r radios) note(p Pos, b Block) bool {
	_, was := r.transmitters[p]
	delete(r.transmitters, p)
	delete(r.receivers, p)
	switch b.Item() {
	case Transmitter:
		r.transmitters[p] = struct{}{}
		return true
	case Receiver:
		r.receivers[p] = struct{}{}
	}
	return was
}
