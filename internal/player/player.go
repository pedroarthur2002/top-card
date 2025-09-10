package player

import "top-card/internal/card"

type Player struct {
    id       int
    userName string
    password string
    wins     int
    losses   int
    inventory []card.Card // Inventário de cartas do jogador
}

func NewPlayer(id int, userName string, password string) Player {
    return Player {
        id:       id,
        userName: userName,
        password: password,
        wins:     0,
        losses:   0,
        inventory: make([]card.Card, 0), // Inicializa inventário vazio
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

// Novos métodos para o sistema de cartas
func (p Player) GetInventory() []card.Card {
    return p.inventory
}

func (p *Player) AddCards(cards []card.Card) {
    p.inventory = append(p.inventory, cards...)
}

func (p Player) GetInventorySize() int {
    return len(p.inventory)
}

// Método para contar cartas por tipo
func (p Player) CountCardsByType() (int, int, int) {
    hydraCount := 0
    quimeraCount := 0
    gorgonaCount := 0
    
    for _, c := range p.inventory {
        switch c.Type {
        case card.HYDRA:
            hydraCount++
        case card.QUIMERA:
            quimeraCount++
        case card.GORGONA:
            gorgonaCount++
        }
    }
    
    return hydraCount, quimeraCount, gorgonaCount
}

// Método para verificar se tem carta específica
func (p Player) HasCardType(cardType string) bool {
    for _, c := range p.inventory {
        if c.Type == cardType {
            return true
        }
    }
    return false
}

// Método para remover uma carta do inventário (para jogar)
func (p *Player) RemoveCard(cardType string) bool {
    for i, c := range p.inventory {
        if c.Type == cardType {
            // Remove a carta do slice
            p.inventory = append(p.inventory[:i], p.inventory[i+1:]...)
            return true
        }
    }
    return false
}