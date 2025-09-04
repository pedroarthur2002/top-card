package match

import (
	"fmt"
	"sync"
	"top-card/Player"
)

// Estrutura para representar uma partida
type Match struct {
	ID      int
	Player1 Player.Player
	Player2 Player.Player
	Status  string // "waiting", "playing", "finished"
	Winner  int    // ID do vencedor, 0 se ainda não há vencedor
}

// Gerenciador de partidas
type MatchManager struct {
	matches    []Match
	nextID     int
	mutex      sync.Mutex
}

// Instância global do gerenciador
var manager = &MatchManager{
	matches: make([]Match, 0),
	nextID:  1,
}

// Função para obter o gerenciador
func GetManager() *MatchManager {
	return manager
}

// Cria uma nova partida com dois jogadores
func (mm *MatchManager) CreateMatch(player1, player2 Player.Player) *Match {
	mm.mutex.Lock()
	defer mm.mutex.Unlock()

	newMatch := Match{
		ID:      mm.nextID,
		Player1: player1,
		Player2: player2,
		Status:  "waiting",
		Winner:  0,
	}

	mm.matches = append(mm.matches, newMatch)
	mm.nextID++

	fmt.Printf("🎮 Nova partida criada! ID: %d - %s vs %s\n", 
		newMatch.ID, player1.GetUserName(), player2.GetUserName())

	return &newMatch
}

// Busca uma partida por ID
func (mm *MatchManager) GetMatch(matchID int) *Match {
	mm.mutex.Lock()
	defer mm.mutex.Unlock()

	for i := range mm.matches {
		if mm.matches[i].ID == matchID {
			return &mm.matches[i]
		}
	}
	return nil
}

// Busca partida onde um jogador está participando
func (mm *MatchManager) GetPlayerMatch(playerID int) *Match {
	mm.mutex.Lock()
	defer mm.mutex.Unlock()

	for i := range mm.matches {
		match := &mm.matches[i]
		if (match.Player1.GetID() == playerID || match.Player2.GetID() == playerID) && 
		   match.Status != "finished" {
			return match
		}
	}
	return nil
}

// Inicia uma partida
func (mm *MatchManager) StartMatch(matchID int) bool {
	mm.mutex.Lock()
	defer mm.mutex.Unlock()

	for i := range mm.matches {
		if mm.matches[i].ID == matchID {
			mm.matches[i].Status = "playing"
			fmt.Printf("🚀 Partida %d iniciada!\n", matchID)
			return true
		}
	}
	return false
}

// Finaliza uma partida definindo o vencedor
func (mm *MatchManager) FinishMatch(matchID int, winnerID int) bool {
	mm.mutex.Lock()
	defer mm.mutex.Unlock()

	for i := range mm.matches {
		if mm.matches[i].ID == matchID {
			mm.matches[i].Status = "finished"
			mm.matches[i].Winner = winnerID
			
			var winnerName string
			if mm.matches[i].Player1.GetID() == winnerID {
				winnerName = mm.matches[i].Player1.GetUserName()
			} else {
				winnerName = mm.matches[i].Player2.GetUserName()
			}
			
			fmt.Printf("🏆 Partida %d finalizada! Vencedor: %s (ID: %d)\n", 
				matchID, winnerName, winnerID)
			return true
		}
	}
	return false
}

// Lista todas as partidas ativas
func (mm *MatchManager) GetActiveMatches() []Match {
	mm.mutex.Lock()
	defer mm.mutex.Unlock()

	var activeMatches []Match
	for _, match := range mm.matches {
		if match.Status != "finished" {
			activeMatches = append(activeMatches, match)
		}
	}
	return activeMatches
}

// Obtém estatísticas das partidas
func (mm *MatchManager) GetStats() (total, waiting, playing, finished int) {
	mm.mutex.Lock()
	defer mm.mutex.Unlock()

	total = len(mm.matches)
	for _, match := range mm.matches {
		switch match.Status {
		case "waiting":
			waiting++
		case "playing":
			playing++
		case "finished":
			finished++
		}
	}
	return
}