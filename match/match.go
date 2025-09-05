package match

import (
	"fmt"
	"sync"
	"top-card/Player"
)

// Estrutura para representar uma partida (MODIFICADA)
type Match struct {
	ID      int
	Player1 Player.Player
	Player2 Player.Player
	Status  string // "waiting", "playing", "finished"
	Winner  int    // ID do vencedor, 0 se ainda não há vencedor
	
	// Novos campos para a lógica do jogo
	CurrentTurn    int  // ID do jogador que deve jogar agora
	Player1Number  *int // Número jogado pelo Player1 (ponteiro para nil = não jogou)
	Player2Number  *int // Número jogado pelo Player2 (ponteiro para nil = não jogou)
	GameStarted    bool // Se o jogo já começou
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
		CurrentTurn:   0,
		Player1Number: nil,
		Player2Number: nil,
		GameStarted:   false,
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

// NOVA FUNÇÃO: Inicia o jogo de uma partida específica
func (mm *MatchManager) StartGame(matchID int) bool {
	mm.mutex.Lock()
	defer mm.mutex.Unlock()

	for i := range mm.matches {
		if mm.matches[i].ID == matchID && mm.matches[i].Status == "playing" {
			mm.matches[i].GameStarted = true
			// Player1 sempre começa
			mm.matches[i].CurrentTurn = mm.matches[i].Player1.GetID()
			fmt.Printf("🎮 Jogo da partida %d iniciado! Turno do Player1 (ID: %d)\n", 
				matchID, mm.matches[i].Player1.GetID())
			return true
		}
	}
	return false
}

// NOVA FUNÇÃO: Processa uma jogada
func (mm *MatchManager) MakeMove(matchID, playerID, number int) (bool, string) {
	mm.mutex.Lock()
	defer mm.mutex.Unlock()

	for i := range mm.matches {
		match := &mm.matches[i]
		if match.ID == matchID {
			// Verificações básicas
			if match.Status != "playing" {
				return false, "Partida não está em andamento"
			}
			
			if !match.GameStarted {
				return false, "Jogo ainda não foi iniciado"
			}
			
			if match.CurrentTurn != playerID {
				return false, "Não é seu turno"
			}

			// Verifica se é o Player1 ou Player2 e se já jogou
			if match.Player1.GetID() == playerID {
				if match.Player1Number != nil {
					return false, "Você já fez sua jogada"
				}
				match.Player1Number = &number
				fmt.Printf("🎯 Player1 (ID: %d) jogou: %d\n", playerID, number)
			} else if match.Player2.GetID() == playerID {
				if match.Player2Number != nil {
					return false, "Você já fez sua jogada"
				}
				match.Player2Number = &number
				fmt.Printf("🎯 Player2 (ID: %d) jogou: %d\n", playerID, number)
			} else {
				return false, "Você não faz parte desta partida"
			}

			// Verifica se ambos jogaram para determinar vencedor
			if match.Player1Number != nil && match.Player2Number != nil {
				return mm.finishGame(match)
			} else {
				// Passa o turno para o outro jogador
				if match.CurrentTurn == match.Player1.GetID() {
					match.CurrentTurn = match.Player2.GetID()
				} else {
					match.CurrentTurn = match.Player1.GetID()
				}
				return true, "Jogada realizada com sucesso! Aguardando o oponente..."
			}
		}
	}
	return false, "Partida não encontrada"
}

// NOVA FUNÇÃO: Finaliza o jogo e determina o vencedor
func (mm *MatchManager) finishGame(match *Match) (bool, string) {
	player1Num := *match.Player1Number
	player2Num := *match.Player2Number
	
	var winnerID int
	var message string
	
	if player1Num > player2Num {
		winnerID = match.Player1.GetID()
		message = fmt.Sprintf("Jogo finalizado! %s venceu com %d contra %d", 
			match.Player1.GetUserName(), player1Num, player2Num)
	} else if player2Num > player1Num {
		winnerID = match.Player2.GetID()
		message = fmt.Sprintf("Jogo finalizado! %s venceu com %d contra %d", 
			match.Player2.GetUserName(), player2Num, player1Num)
	} else {
		// Empate - vamos considerar Player1 como vencedor por padrão
		winnerID = match.Player1.GetID()
		message = fmt.Sprintf("Empate! Ambos jogaram %d. %s vence por começar primeiro", 
			player1Num, match.Player1.GetUserName())
	}
	
	match.Status = "finished"
	match.Winner = winnerID
	
	fmt.Printf("🏆 %s\n", message)
	return true, message
}

// NOVA FUNÇÃO: Verifica se é o turno do jogador
func (mm *MatchManager) IsPlayerTurn(matchID, playerID int) bool {
	mm.mutex.Lock()
	defer mm.mutex.Unlock()

	for i := range mm.matches {
		if mm.matches[i].ID == matchID {
			return mm.matches[i].CurrentTurn == playerID && mm.matches[i].GameStarted
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