package gateway

import (
	"testing"

	pb "scabbage/internal/pb/scabbage/v1"
	"scabbage/internal/world"
)

func requestFar(columns ...*pb.ColumnPos) *pb.ClientMessage {
	return &pb.ClientMessage{Body: &pb.ClientMessage_RequestFar{RequestFar: &pb.RequestFar{Columns: columns}}}
}

func TestFarLandSketchesColumnsInsideTheBorder(t *testing.T) {
	ts := startServer(t)
	c := ts.joinAdmin(t)
	send(t, c, requestFar(&pb.ColumnPos{X: 1, Z: -2}, &pb.ColumnPos{X: 2}, &pb.ColumnPos{Z: maxFarRange + 1}))
	land := recv(t, c).GetFarLand()
	if land == nil || len(land.Tiles) != 1 || land.Tiles[0].Pos.X != 1 || land.Tiles[0].Pos.Z != -2 {
		t.Fatalf("far land = %v", land)
	}
	want, _ := ts.world.FarTile(1, -2)
	tile := land.Tiles[0]
	n := world.FarCells * world.FarCells
	if len(tile.Tops) != n || len(tile.Blocks) != n || tile.Tops[9] != uint32(want.Tops[9]) || tile.Blocks[9] != uint32(want.Blocks[9]) {
		t.Fatalf("tile = %v", tile)
	}
}

func TestOversizedFarRequestIsRejected(t *testing.T) {
	ts := startServer(t)
	c := ts.joinAdmin(t)
	columns := make([]*pb.ColumnPos, maxChunksPerRequest+1)
	for i := range columns {
		columns[i] = &pb.ColumnPos{X: int32(i)}
	}
	send(t, c, requestFar(columns...))
	expectDisconnect(t, c, pb.DisconnectCode_DISCONNECT_CODE_PROTOCOL)
}
