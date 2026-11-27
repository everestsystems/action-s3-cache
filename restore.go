package main

type objectStore interface {
	Exists(key string) (bool, error)
	Download(key string) error
}

func restore(store objectStore, key string) (bool, error) {
	exists, err := store.Exists(key)
	if err != nil || !exists {
		return false, err
	}

	return true, store.Download(key)
}
