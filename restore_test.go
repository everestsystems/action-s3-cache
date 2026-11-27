package main

import "testing"

type stubStore struct {
	keys map[string]bool
}

func (s stubStore) Exists(key string) (bool, error) {
	return s.keys[key], nil
}

func (s stubStore) Download(string) error {
	return nil
}

func TestRestoreReportsHitWhenKeyExists(t *testing.T) {
	t.Parallel()

	store := stubStore{keys: map[string]bool{"cache.tgz": true}}

	hit, err := restore(store, "cache.tgz")
	if err != nil {
		t.Fatal(err)
	}

	if !hit {
		t.Errorf("got cache hit %t, want %t", hit, true)
	}
}
