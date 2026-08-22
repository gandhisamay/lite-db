package store

type Memtable struct {
	data map[string]entry
	size int
}

type entry struct {
	value   string
	deleted bool
}

func New() *Memtable {
	return &Memtable{
		data: make(map[string]entry),
		size: 0,
	}
}

// Put adds or replaces the value for key.
func (mt *Memtable) Put(key string, value string) {
	if _, exists := mt.data[key]; !exists {
		mt.size++
	}

	mt.data[key] = entry{value: value}
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
	if _, exists := mt.data[key]; !exists {
		mt.size++
	}

	mt.data[key] = entry{deleted: true}
}

// Size returns the number of distinct keys represented by the memtable.
// Deleted keys are included because their tombstones are represented too.
func (mt *Memtable) Size() int {
	return mt.size
}
