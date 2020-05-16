package web

import (
	"fmt"
	"hash/fnv"
	"io"
	"net/http"

	"github.com/YuriyNasretdinov/distribkv/db"
)

// Server contains HTTP method handlers to be used for the database.
type Server struct {
	db         *db.Database
	shardIdx   int
	shardCount int
	addrs      map[int]string
}

// NewServer creates a new instance with HTTP handlers to be used to get and set values.
func NewServer(db *db.Database, shardIdx, shardCount int, addrs map[int]string) *Server {
	return &Server{
		db:         db,
		shardIdx:   shardIdx,
		shardCount: shardCount,
		addrs:      addrs,
	}
}

func (s *Server) getShard(key string) int {
	h := fnv.New64()
	h.Write([]byte(key))
	return int(h.Sum64() % uint64(s.shardCount))
}

func (s *Server) redirect(shard int, w http.ResponseWriter, r *http.Request) {
	url := "http://" + s.addrs[shard] + r.RequestURI
	fmt.Fprintf(w, "redirecting from shard %d to shard %d (%q)\n", s.shardIdx, shard, url)

	resp, err := http.Get(url)
	if err != nil {
		w.WriteHeader(500)
		fmt.Fprintf(w, "Error redirecting the request: %v", err)
		return
	}
	defer resp.Body.Close()

	io.Copy(w, resp.Body)
}

// GetHandler handles read requests from the database.
func (s *Server) GetHandler(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	key := r.Form.Get("key")

	shard := s.getShard(key)
	value, err := s.db.GetKey(key)

	if shard != s.shardIdx {
		s.redirect(shard, w, r)
		return
	}

	fmt.Fprintf(w, "Shard = %d, current shard = %d, addr = %q, Value = %q, error = %v", shard, s.shardIdx, s.addrs[shard], value, err)
}

// SetHandler handles write requests from the database.
func (s *Server) SetHandler(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	key := r.Form.Get("key")
	value := r.Form.Get("value")

	shard := s.getShard(key)
	if shard != s.shardIdx {
		s.redirect(shard, w, r)
		return
	}

	err := s.db.SetKey(key, []byte(value))
	fmt.Fprintf(w, "Error = %v, shardIdx = %d, current shard = %d", err, shard, s.shardIdx)
}
