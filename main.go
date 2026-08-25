package main

import (
	"fmt"
	"os"

	"github.com/gandhisamay/lite-db/database"
)

// Operation is the user-facing command representation used by the CLI.
type Operation string

const (
	Get    Operation = "GET"
	Set    Operation = "SET"
	Delete Operation = "DELETE"
)

func (operation Operation) databaseType() database.OperationType {
	switch operation {
	case Get:
		return database.OpGet
	case Set:
		return database.OpSet
	case Delete:
		return database.OpDelete
	default:
		return database.OpInvalid
	}
}

func main() {
	// this is the main function
	// we provide the

	command := Operation(os.Args[1])
	key := os.Args[2]
	var value string

	if len(os.Args) == 4 {
		value = os.Args[3]
	}

	db := database.Start()

	res, valid := db.Perform(command.databaseType(), key, value)

	if command == Get && !valid {
		fmt.Println("Key not found in the database")
		return
	}

	if !valid {
		fmt.Println("error occurred")
		return
	}

	fmt.Printf("%s %s - %s\n", command, key, res)
}
