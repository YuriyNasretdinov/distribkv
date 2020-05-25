package db_test

import (
	"bytes"
	"io/ioutil"
	"os"
	"testing"

	"github.com/YuriyNasretdinov/distribkv/db"
)

func TestGetSet(t *testing.T) {
	f, err := ioutil.TempFile(os.TempDir(), "kvdb")
	if err != nil {
		t.Fatalf("Could not create temp file: %v", err)
	}
	name := f.Name()
	f.Close()
	defer os.Remove(name)

	db, closeFunc, err := db.NewDatabase(name)
	if err != nil {
		t.Fatalf("Could not create a new database: %v", err)
	}
	defer closeFunc()

	if err := db.SetKey("party", []byte("Great")); err != nil {
		t.Fatalf("Could not write key: %v", err)
	}

	value, err := db.GetKey("party")
	if err != nil {
		t.Fatalf(`Could not get the key "party": %v`, err)
	}

	if !bytes.Equal(value, []byte("Great")) {
		t.Errorf(`Unexpected value for key "party": got %q, want %q`, value, "Great")
	}
}

func setKey(t *testing.T, d *db.Database, key, value string) {
	t.Helper()

	if err := d.SetKey(key, []byte(value)); err != nil {
		t.Fatalf("SetKey(%q, %q) failed: %v", key, value, err)
	}
}

func getKey(t *testing.T, d *db.Database, key string) string {
	t.Helper()

	value, err := d.GetKey(key)
	if err != nil {
		t.Fatalf("GetKey(%q) failed: %v", key, err)
	}

	return string(value)
}

func TestDeleteExtraKeys(t *testing.T) {
	f, err := ioutil.TempFile(os.TempDir(), "kvdb")
	if err != nil {
		t.Fatalf("Could not create temp file: %v", err)
	}
	name := f.Name()
	f.Close()
	defer os.Remove(name)

	db, closeFunc, err := db.NewDatabase(name)
	if err != nil {
		t.Fatalf("Could not create a new database: %v", err)
	}
	defer closeFunc()

	setKey(t, db, "party", "Great")
	setKey(t, db, "us", "CapitalistPigs")

	if err := db.DeleteExtraKeys(func(name string) bool { return name == "us" }); err != nil {
		t.Fatalf("Could not delete extra keys: %v", err)
	}

	if value := getKey(t, db, "party"); value != "Great" {
		t.Errorf(`Unexpected value for key "party": got %q, want %q`, value, "Great")
	}

	if value := getKey(t, db, "us"); value != "" {
		t.Errorf(`Unexpected value for key "us": got %q, want %q`, value, "")
	}
}
