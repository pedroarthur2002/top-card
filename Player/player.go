package main

import(
	"fmt"
)

type Player struct {
	id int 
	userName string
	password string
	wins int
	losses int
}

func registerPlayer (id int, userName string, password string) Player{
	return Player {
		id: id,
		userName: userName,
		password: password,
		wins: 0,
		losses: 0,
	}
}

/*
func loginPlayer (userName string, password string,) bool{
	
}

*/
