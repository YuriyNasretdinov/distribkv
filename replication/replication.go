package replication

import (
	"bytes"
	"encoding/json"
	"errors"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"time"

	"github.com/YuriyNasretdinov/distribkv/db"
)

// NextKeyValue contains the response for GetNextKeyForReplication.
type NextKeyValue struct {
	Key   string
	Value string
	Err   error
}

type client struct {
	db         *db.Database
	leaderAddr string
}

// ClientLoop continuously downloads new keys from the master and applies them.
func ClientLoop(db *db.Database, leaderAddr string) {
	c := &client{db: db, leaderAddr: leaderAddr}
	for {
		present, err := c.loop()
		if err != nil {
			log.Printf("Loop error: %v", err)
			time.Sleep(time.Second)
			continue
		}

		if !present {
			time.Sleep(time.Millisecond * 100)
		}
	}
}

func (c *client) loop() (present bool, err error) {
	resp, err := http.Get("http://" + c.leaderAddr + "/next-replication-key")
	if err != nil {
		return false, err
	}

	var res NextKeyValue
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return false, err
	}
	defer resp.Body.Close()

	if res.Err != nil {
		return false, err
	}

	if res.Key == "" {
		return false, nil
	}

	if err := c.db.SetKeyOnReplica(res.Key, []byte(res.Value)); err != nil {
		return false, err
	}

	if err := c.deleteFromReplicationQueue(res.Key, res.Value); err != nil {
		log.Printf("DeleteKeyFromReplication failed: %v", err)
	}

	return true, nil
}

func (c *client) deleteFromReplicationQueue(key, value string) error {
	u := url.Values{}
	u.Set("key", key)
	u.Set("value", value)

	log.Printf("Deleting key=%q, value=%q from replication queue on %q", key, value, c.leaderAddr)

	resp, err := http.Get("http://" + c.leaderAddr + "/delete-replication-key?" + u.Encode())
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	result, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	if !bytes.Equal(result, []byte("ok")) {
		return errors.New(string(result))
	}

	return nil
}
