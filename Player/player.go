package Player

type Player struct {
	id       int 
	userName string
	password string
	wins     int
	losses   int
}

func NewPlayer(id int, userName string, password string) Player {
	return Player {
		id:       id,
		userName: userName,
		password: password,
		wins:     0,
		losses:   0,
	}
}

// Métodos getters públicos
func (p Player) GetUserName() string {
	return p.userName
}

func (p Player) GetPassword() string {
	return p.password
}

func (p Player) GetID() int {
	return p.id
}


/*
func loginPlayer (userName string, password string,) bool{
	
}

*/
