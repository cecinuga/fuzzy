package utils

import "fmt"

type ResponseMsg struct {
	Status 		string
	FuzzValue 	string
	Time 		string
	ResMsg 		string
}

func Log(res ResponseMsg){
	init := "[+]"
	if res.Status[0] != '2' {
		init = "[!]"
	}

	fmt.Printf("%v %v %v <val: %v> %v\n", init, res.Time, res.Status, res.FuzzValue, res.ResMsg)
}