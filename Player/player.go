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

// Métodos getters públicos existentes
func (p Player) GetUserName() string {
	return p.userName
}

func (p Player) GetPassword() string {
	return p.password
}

func (p Player) GetID() int {
	return p.id
}

// Novos métodos para estatísticas
func (p Player) GetWins() int {
	return p.wins
}

func (p Player) GetLosses() int {
	return p.losses
}

func (p *Player) AddWin() {
	p.wins++
}

func (p *Player) AddLoss() {
	p.losses++
}

func (p Player) GetWinRate() float64 {
	totalGames := p.wins + p.losses
	if totalGames == 0 {
		return 0.0
	}
	return float64(p.wins) / float64(totalGames) * 100
}