package cleanup

import (
	"context"
	"log"
	"time"

	"chatemp/internal/config"
	"chatemp/internal/models"
	"chatemp/internal/storage"
	"chatemp/internal/store"
	"chatemp/internal/ws"
)

func Run(ctx context.Context, cfg config.Config, st *store.Store, hub *ws.Hub, fs *storage.Storage) {
	go runMediaSweep(ctx, cfg, st, hub, fs)
	go runRoomPurge(ctx, cfg, st, hub, fs)
}

// runMediaSweep recovers ephemeral media deletions lost on restart.
// Precise deletion of consumed media happens via in-memory timers; this
// sweep is only a safety net and runs infrequently.
func runMediaSweep(ctx context.Context, cfg config.Config, st *store.Store, hub *ws.Hub, fs *storage.Storage) {
	sweep := func() {
		cutoff := time.Now().UTC().Add(-cfg.EphemeralTTL)
		msgs, err := st.ConsumedMedia(ctx, cutoff)
		if err != nil {
			log.Printf("media sweep: %v", err)
			return
		}
		for _, m := range msgs {
			DeleteMedia(ctx, st, hub, fs, m)
		}
	}
	sweep()
	t := time.NewTicker(cfg.MediaSweep)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			sweep()
		}
	}
}

// runRoomPurge physically deletes expired rooms (rows + files). Access is
// already blocked lazily per request; this only reclaims disk space.
func runRoomPurge(ctx context.Context, cfg config.Config, st *store.Store, hub *ws.Hub, fs *storage.Storage) {
	purge := func() {
		rooms, err := st.ExpiredRooms(ctx, time.Now().UTC())
		if err != nil {
			log.Printf("room purge: %v", err)
			return
		}
		for _, room := range rooms {
			hub.Broadcast(room.ID, ws.Payload{Type: "room_expired"})
			hub.CloseRoom(room.ID)
			paths, err := st.MediaPathsByRoom(ctx, room.ID)
			if err == nil {
				for _, p := range paths {
					fs.Delete(p)
				}
			}
			fs.DeleteRoom(room.ID)
			if err := st.DeleteRoom(ctx, room.ID); err != nil {
				log.Printf("delete room %s: %v", room.ID, err)
			}
		}
	}
	purge()
	t := time.NewTicker(cfg.RoomPurge)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			purge()
		}
	}
}

// DeleteMedia removes an ephemeral media message: file + row + broadcast.
func DeleteMedia(ctx context.Context, st *store.Store, hub *ws.Hub, fs *storage.Storage, m models.Message) {
	if m.MediaPath != "" {
		if err := fs.Delete(m.MediaPath); err != nil {
			log.Printf("delete file %s: %v", m.MediaPath, err)
		}
	}
	if err := st.DeleteMessage(ctx, m.ID); err != nil {
		log.Printf("delete message %s: %v", m.ID, err)
		return
	}
	hub.Broadcast(m.RoomID, ws.Payload{Type: "media_deleted", MessageID: m.ID})
}
