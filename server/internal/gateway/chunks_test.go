package gateway

import (
	"testing"

	"github.com/coder/websocket"

	pb "scabbage/internal/pb/scabbage/v1"
	"scabbage/internal/world"
)

func requestChunks(positions ...*pb.ChunkPos) *pb.ClientMessage {
	return &pb.ClientMessage{Body: &pb.ClientMessage_RequestChunks{
		RequestChunks: &pb.RequestChunks{Chunks: positions},
	}}
}

func TestWelcomeDescribesTheWorld(t *testing.T) {
	ts := startServer(t)
	_ = ts.joinAdmin(t).Close(websocket.StatusNormalClosure, "")
	waitPlayers(t, ts.hub, 0)

	c := dial(t, ts.url)
	send(t, c, hello(secret(1), nil))
	info := recv(t, c).GetWelcome().GetWorld()
	if info == nil || info.Radius != 64 || info.Height != world.MaxY || info.Spawn.GetY() <= 0 {
		t.Fatalf("world info = %v", info)
	}
	if len(info.Blocks) != len(world.Blocks()) || info.Blocks[world.Glass].Name != "glass" || !info.Blocks[world.Glass].Transparent {
		t.Fatalf("block table = %v", info.Blocks)
	}
}

func TestChunkRequestGetsOneBatchInOrder(t *testing.T) {
	ts := startServer(t)
	c := ts.joinAdmin(t)
	send(t, c, requestChunks(&pb.ChunkPos{X: 0, Y: 2, Z: 0}, &pb.ChunkPos{X: -1, Y: 0, Z: 3}, &pb.ChunkPos{X: 5, Y: 0, Z: 0}))
	batch := recv(t, c).GetChunkBatch()
	if batch == nil || len(batch.Chunks) != 3 || batch.Chunks[1].Pos.X != -1 || batch.Chunks[1].Pos.Z != 3 {
		t.Fatalf("batch = %v", batch)
	}
	for _, ch := range batch.Chunks {
		if _, err := world.Decode(ch.Data); err != nil {
			t.Fatalf("chunk %v: %v", ch.Pos, err)
		}
	}
	if bottom, _ := world.Decode(batch.Chunks[1].Data); bottom.Get(0, 0, 0) != world.Bedrock {
		t.Fatal("the bottom of a column should be bedrock")
	}
}

func TestChunksFarFromThePlayerAreLeftOut(t *testing.T) {
	ts := startServer(t)
	c := ts.joinAdmin(t)
	far := int32(maxChunkRange + 1)
	send(t, c, requestChunks(&pb.ChunkPos{X: far}, &pb.ChunkPos{X: 1, Z: 1}, &pb.ChunkPos{Z: -far}))
	batch := recv(t, c).GetChunkBatch()
	if batch == nil || len(batch.Chunks) != 1 || batch.Chunks[0].Pos.X != 1 {
		t.Fatalf("batch = %v", batch)
	}
}

func TestOversizedChunkRequestIsRejected(t *testing.T) {
	ts := startServer(t)
	c := ts.joinAdmin(t)
	positions := make([]*pb.ChunkPos, maxChunksPerRequest+1)
	for i := range positions {
		positions[i] = &pb.ChunkPos{X: int32(i)}
	}
	send(t, c, requestChunks(positions...))
	expectDisconnect(t, c, pb.DisconnectCode_DISCONNECT_CODE_PROTOCOL)
}

func TestConsoleCannotRequestChunks(t *testing.T) {
	ts := startServer(t)
	_ = ts.joinAdmin(t).Close(websocket.StatusNormalClosure, "")
	waitPlayers(t, ts.hub, 0)

	c := dial(t, ts.url)
	msg := hello(secret(1), nil)
	msg.GetHello().Console = true
	send(t, c, msg)
	if w := recv(t, c).GetWelcome(); w == nil || w.World != nil {
		t.Fatalf("console welcome = %v", w)
	}
	recv(t, c) // pending joins
	recv(t, c) // pending art
	send(t, c, requestChunks(&pb.ChunkPos{}))
	expectDisconnect(t, c, pb.DisconnectCode_DISCONNECT_CODE_PROTOCOL)
}
