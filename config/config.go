package config

// Shard describes a shard that holds the appropriate set of keys.
// Each shard has unique set of keys.
type Shard struct {
	Name    string
	Idx     int
	Address string
}

// Config describes the sharding config.
type Config struct {
	Shards []Shard
}
