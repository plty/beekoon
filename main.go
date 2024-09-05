package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
)

type VersionedOutput struct {
	Version int    `json:"version"`
	Body    string `json:"body"`
}

type FetcherState struct {
	state *VersionedOutput
	cond  *sync.Cond

	// metrics
	members int64
}

func NewFetcher() *FetcherState {
	fs := FetcherState{
		state: &VersionedOutput{
			Version: 0,
			Body:    "Hello hello!",
		},
		cond: sync.NewCond(&sync.Mutex{}),
	}
	go func() {
		version := 0
		for {
			{
				// pointer reassignment is atomic, modifying struct is not.
				fs.state = &VersionedOutput{
					Version: version,
					Body:    fmt.Sprintf("body updated %d", version),
				}
				fs.cond.Broadcast()

				fmt.Println("ver:", version, "mem:", atomic.LoadInt64(&fs.members))
			}
			time.Sleep(250 * time.Millisecond)
			version++
		}
	}()

	return &fs
}

func MakeFeed(fs *FetcherState) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt64(&fs.members, 1)
		defer atomic.AddInt64(&fs.members, -1)

		ctx := r.Context()

		upgrader := websocket.Upgrader{}
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		defer conn.Close()
		last := 0
		for {
			if fs.state.Version == last {
				fs.cond.L.Lock()
				if fs.state.Version == last {
					fs.cond.Wait()
				}
				fs.cond.L.Unlock()
			}
			if ctx.Done() == nil {
				return
			}

			last = fs.state.Version
			if conn.WriteJSON(fs.state.Body) != nil {
				break
			}
		}
	})
}

func MakeSimple(fs *FetcherState) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		err := json.NewEncoder(w).Encode(fs.state)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	}
}

func main() {
	fs := NewFetcher()
	router := mux.NewRouter()
	router.HandleFunc("/", MakeSimple(fs))
	router.HandleFunc("/ws", MakeFeed(fs))
	http.ListenAndServe(":8080", router)
}
