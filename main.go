package main

import (
	"fmt"
	"os"

	"com.db.beginner/database"
)

func main() {
	// this is the main function
	// we provide the

	command := os.Args[1]
	key := os.Args[2]
	var value string

	if len(os.Args) == 4 {
		value = os.Args[3]
	}

	db := database.Start()

	res, valid := db.Perform(command, key, value)

	if !valid {
		fmt.Println("error occurred")
	}

	fmt.Printf("%s %s - %s\n", command, key, res)
}
