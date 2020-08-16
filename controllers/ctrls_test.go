package controllers

import (
	"fmt"
	"os"
)

func init() {
	// change dir to repo root
	os.Chdir("../")
	newDir, err := os.Getwd()
	if err != nil {
		fmt.Printf("Error changing directory")
	}
	fmt.Printf("Current Working Direcotry: %s\n", newDir)
}
