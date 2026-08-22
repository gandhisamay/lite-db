// Package database contains all db related logic
package database

import (
	"fmt"
	"log"
	"strings"

	"com.db.beginner/store"
)

type Database struct {
	data    map[string]string
	store   *store.Store
	isReady bool
}

type Operation string

const (
	GET    Operation = "GET"
	SET    Operation = "SET"
	DELETE Operation = "DELETE"
	PUT    Operation = "PUT"
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

	operation := Operation(logArr[0])

	switch operation {
	case PUT:
		key := logArr[1]
		value := logArr[2]
		db.data[key] = value
	case SET:
		key := logArr[1]
		value := logArr[2]
		db.data[key] = value
	case DELETE:
		key := logArr[1]
		delete(db.data, key)
	}
}

func (db *Database) Close() {
	db.store.Close()
	db.isReady = false
}

func (db *Database) Perform(command string, values ...string) (string, bool) {
	operation := Operation(command)

	switch operation {
	case GET:
		return db.get(values[0])
	case SET:
		return db.set(values[0], values[1])
	case PUT:
		return db.put(values[0], values[1])
	case DELETE:
		return db.delete(values[0])
	default:
		fmt.Println("Invalid operation, valid operations are GET, SET, PUT, DELETE")
		return emptyString, false
	}
}

func (db *Database) get(key string) (string, bool) {
	value, exists := db.data[key]
	return value, exists
}

func (db *Database) set(key string, value string) (string, bool) {
	// write to the file first
	log := fmt.Sprintf("SET %s %s\n", key, value)

	err := db.store.Append(log)
	if err != nil {
		fmt.Println(err)
		fmt.Println("SET operation failed")
		return emptyString, false
	}

	// if that doesn't fail, we write to the map
	db.data[key] = value
	return key, true
}

func (db *Database) put(key string, value string) (string, bool) {
	log := fmt.Sprintf("PUT %s %s\n", key, value)

	err := db.store.Append(log)
	if err != nil {
		fmt.Println(err)
		fmt.Println("fsync failed for the SET operation")
		return emptyString, false
	}

	// if that doesn't fail, we write to the map
	db.data[key] = value
	return key, true
}

func (db *Database) delete(key string) (string, bool) {
	log := fmt.Sprintf("DELETE %s\n", key)
	err := db.store.Append(log)
	if err != nil {
		fmt.Println(err)
		fmt.Println("delete operation failed")
		return emptyString, false
	}

	// if that doesn't fail, we write to the map
	delete(db.data, key)
	return key, true
}
