// Package memtable provides the in-memory table used by the storage engine.
package memtable

import "sort"

type Memtable struct {
	data map[string]entry
	size int
}

type entry struct {
	value   string
	deleted bool
}

type Record struct {
	Key     string
	Value   string
	Deleted bool
}

const MaxMemTableSize int32 = 4 * 1024 * 1024

// NewMemtable creates an empty memtable.
func NewMemtable() *Memtable {
	return &Memtable{data: make(map[string]entry)}
}

// Set adds or replaces the value for key.
func (mt *Memtable) Set(key string, value string) {
	if old, exists := mt.data[key]; exists {
		mt.size -= len(key) + len(old.value)
		if old.deleted {
			mt.size--
		}
	}

	mt.data[key] = entry{value: value}
	mt.size += len(key) + len(value)
}

// Get returns the value for key. A missing key and a deleted key both return
// false.
func (mt *Memtable) Get(key string) (string, bool) {
	entry, exists := mt.data[key]
	if !exists || entry.deleted {
		return "", false
	}

	return entry.value, true
}

// Delete records a tombstone for key. Keeping the tombstone is important for
// a memtable because it prevents an older value from being resurrected when
// the table is later flushed or merged with older data.
func (mt *Memtable) Delete(key string) {
	if old, exists := mt.data[key]; exists {
		mt.size -= len(key) + len(old.value)
		if old.deleted {
			mt.size--
		}
	}

	mt.data[key] = entry{deleted: true}
	mt.size += len(key) + 1
}

// Size returns the number of bytes represented by the memtable, including
// tombstone overhead.
func (mt *Memtable) Size() int {
	return mt.size
}

func (mt *Memtable) SortedEntries() []Record {
	entries := make([]Record, len(mt.data))

	i := 0

	for key, entry := range mt.data {
		entries[i] = Record{
			Key:     key,
			Value:   entry.value,
			Deleted: entry.deleted,
		}

		i++
	}

	// now we have to sort this
	sort.Slice(entries, func(i int, j int) bool {
		return entries[i].Key < entries[j].Key
	})

	return entries
}
