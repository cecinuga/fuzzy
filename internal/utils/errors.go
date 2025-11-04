package utils

import "fmt"

type KeyNotFoundError struct {
	Key string
	Msg string
}

func (e KeyNotFoundError) Error() string {
	return fmt.Sprintf("%v. Key %v not found. ", e.Key, e.Msg)
}

type ObjectNotInitialized struct {
	Msg string
}

func (e ObjectNotInitialized) Error() string {
	return "Object not initialized yet."
}