package utils

import (
	"fmt"
)

type ResponseMsg struct {
	Status 		string
	QueryParams string
	FuzzValue 	string
	Time 		string
	Error 		bool
}

func Log(res ResponseMsg){
	init := "[#]"

	if res.Error { init="[!]" }

	fmt.Printf("%v %v %v %v %v\n", init, res.Time, res.Status, res.FuzzValue, res.QueryParams)
}