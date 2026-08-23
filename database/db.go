// Package database contains all db related logic
package database

import (
	"fmt"
	"log"
	"strings"

	"github.com/gandhisamay/lite-db/store"
)

type Database struct {
	data    map[string]string
	store   *store.Store
	isReady bool
}

type OperationType = store.OperationType

const (
	OpGet     = store.OpGet
	OpSet     = store.OpSet
	OpDelete  = store.OpDelete
	OpInvalid = store.OpInvalid
)

const emptyString string = ""

func Start() *Database {
	// prepares the database and returns the database object
	// read all the data from the file
	dbStore, err := store.Open()
	if err != nil {
		log.Fatalln(err)
	}

	db := &Database{
		data:    make(map[string]string),
		store:   dbStore,
		isReady: true,
	}

	err = db.store.Replay(db.applyRecord)
	if err != nil {
		db.store.Close()
		log.Fatalln("wal replay failed:", err)
	}

	return db
}

func (db *Database) applyRecord(payload []byte) {
	logArr := strings.Fields(string(payload))
	if len(logArr) == 0 {
		return
	}

	switch logArr[0] {
	case "SET":
		if len(logArr) < 3 {
			return
		}
		key := logArr[1]
		value := logArr[2]
		db.data[key] = value
	case "DELETE":
		if len(logArr) < 2 {
			return
		}
		key := logArr[1]
		delete(db.data, key)
	}
}

func (db *Database) Close() {
	db.store.Close()
	db.isReady = false
}

func (db *Database) Perform(operation OperationType, values ...string) (string, bool) {
	switch operation {
	case OpGet:
		return db.get(values[0])
	case OpSet:
		return db.set(values[0], values[1])
	case OpDelete:
		return db.delete(values[0])
	default:
		fmt.Println("Invalid operation, valid operations are GET, SET, DELETE")
		return emptyString, false
	}
}

func (db *Database) get(key string) (string, bool) {
	value, exists := db.data[key]
	return value, exists
}

func (db *Database) set(key string, value string) (string, bool) {
	// write to the file first
	err := db.store.Append(store.WriteRequest{
		Operation: store.OpSet,
		Key:       key,
		Value:     value,
	})
	if err != nil {
		fmt.Println(err)
		fmt.Println("SET operation failed")
		return emptyString, false
	}

	// if that doesn't fail, we write to the map
	db.data[key] = value
	return key, true
}

func (db *Database) delete(key string) (string, bool) {
	err := db.store.Append(store.WriteRequest{
		Operation: store.OpDelete,
		Key:       key,
	})
	if err != nil {
		fmt.Println(err)
		fmt.Println("delete operation failed")
		return emptyString, false
	}

	// if that doesn't fail, we write to the map
	delete(db.data, key)
	return key, true
}
