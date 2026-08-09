// Package database contains all db related logic
package database

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"strings"
)

type Database struct {
	data        map[string]string
	walFile     *os.File
	walFilePath string
	isReady     bool
}

type Operation string

const (
	GET    Operation = "GET"
	SET    Operation = "SET"
	DELETE Operation = "DELETE"
	PUT    Operation = "PUT"
)

const EMPTY_STRING string = ""

func Start() *Database {
	// prepares the database and returns the database object
	// read all the data from the file
	walFilePath := "data.wal"

	file, err := os.OpenFile(walFilePath, os.O_RDWR|os.O_APPEND|os.O_CREATE, 0o644)
	if err != nil {
		log.Fatalln("failed to read the wal file")
	}

	// now we read from the file, and load everything into the data map
	scanner := bufio.NewScanner(file)
	data := make(map[string]string, 0)

	for scanner.Scan() {
		logArr := strings.Fields(scanner.Text())

		operation := Operation(logArr[0])

		switch operation {
		case PUT:
			key := logArr[1]
			value := logArr[2]
			data[key] = value
		case SET:
			key := logArr[1]
			value := logArr[2]
			data[key] = value
		case DELETE:
			key := logArr[1]
			delete(data, key)
		}
	}

	if err := scanner.Err(); err != nil {
		log.Fatalln("failed to read the wal file")
	}

	return &Database{
		data:        data,
		walFile:     file,
		walFilePath: walFilePath,
		isReady:     true,
	}
}

func (db *Database) Close() {
	db.walFile.Close()
	db.isReady = false
}

func (db *Database) Perform(command string, values ...string) (string, bool) {
	operation := Operation(command)

	switch operation {
	case GET:
		// perform the get operation now
		return db.get(values[0])
	case SET:
		// perform the set operation now
		return db.set(values[0], values[1])
	case PUT:
		// perform the put operation now
		return db.put(values[0], values[1])
	case DELETE:
		// perform the delete operation now
		return db.delete(values[0])
	default:
		fmt.Println("Invalid operation, valid operations are GET, SET, PUT, DELETE")
		return EMPTY_STRING, false
	}
}

func (db *Database) get(key string) (string, bool) {
	value, exists := db.data[key]
	return value, exists
}

func (db *Database) set(key string, value string) (string, bool) {
	// write to the file first
	log := fmt.Sprintf("SET %s %s\n", key, value)
	_, err := db.walFile.WriteString(log)
	if err != nil {
		fmt.Println(err)
		fmt.Println("SET operation failed")
		return EMPTY_STRING, false
	}

	// if that doesn't fail, we write to the map
	db.data[key] = value
	return key, true
}

func (db *Database) put(key string, value string) (string, bool) {
	log := fmt.Sprintf("PUT %s %s\n", key, value)
	_, err := db.walFile.WriteString(log)
	if err != nil {
		fmt.Println("put operation failed")
		return EMPTY_STRING, false
	}

	// if that doesn't fail, we write to the map
	db.data[key] = value
	return key, true
}

func (db *Database) delete(key string) (string, bool) {
	log := fmt.Sprintf("DELETE %s\n", key)
	_, err := db.walFile.WriteString(log)
	if err != nil {
		fmt.Println("delete operation failed")
		return EMPTY_STRING, false
	}

	// if that doesn't fail, we write to the map
	delete(db.data, key)
	return key, true
}
