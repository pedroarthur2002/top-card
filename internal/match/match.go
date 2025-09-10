package match

import (
	"fmt"
	"sync"
	"top-card/internal/player"
	"top-card/internal/card"
)

// Estrutura para representar uma partida (MODIFICADA)
type Match struct {
	ID      int
	Player1 *player.Player
	Player2 *player.Player
	Status  string // "waiting", "playing", "finished"
	Winner  int    // ID do vencedor, 0 se ainda não há vencedor
	
	// Campos para a lógica do jogo de cartas
	CurrentTurn    int        // ID do jogador que deve jogar agora
	Player1Card    *card.Card // Carta jogada pelo Player1 (ponteiro para nil = não jogou)
	Player2Card    *card.Card // Carta jogada pelo Player2 (ponteiro para nil = não jogou)
	GameStarted    bool       // Se o jogo já começou
	GameType       string     // "cards" para jogo de cartas, "numbers" para números
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
func (mm *MatchManager) CreateMatch(player1, player2 *player.Player) *Match {
    mm.mutex.Lock()
    defer mm.mutex.Unlock()

    newMatch := Match{
        ID:            mm.nextID,
        Player1:       player1, 
        Player2:       player2,  
        Status:        "waiting",
        Winner:        0,
        CurrentTurn:   0,
        Player1Card:   nil,
        Player2Card:   nil,
        GameStarted:   false,
        GameType:      "cards",
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

// Força vitória por abandono/desconexão
func (mm *MatchManager) ForceWin(matchID int, winnerID int) bool {
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
			
			fmt.Printf("🏆 Partida %d finalizada por abandono! Vencedor: %s (ID: %d)\n", 
				matchID, winnerName, winnerID)
			return true
		}
	}
	return false
}

// Cancela uma partida
func (mm *MatchManager) CancelMatch(matchID int) bool {
	mm.mutex.Lock()
	defer mm.mutex.Unlock()

	for i := range mm.matches {
		if mm.matches[i].ID == matchID {
			mm.matches[i].Status = "cancelled"
			
			fmt.Printf("❌ Partida %d cancelada\n", matchID)
			return true
		}
	}
	return false
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

// Inicia o jogo de uma partida específica
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

// Processa uma jogada com carta
func (mm *MatchManager) MakeCardMove(matchID, playerID int, cardType string) (bool, string) {
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

			// Verifica se é uma carta válida
			if cardType != card.HYDRA && cardType != card.QUIMERA && cardType != card.GORGONA {
				return false, "Tipo de carta inválido"
			}

			// NOVA VALIDAÇÃO: Verifica se o jogador possui a carta
			var currentPlayer *player.Player
			if match.Player1.GetID() == playerID {
				if match.Player1Card != nil {
					return false, "Você já fez sua jogada"
				}
				currentPlayer = match.Player1
			} else if match.Player2.GetID() == playerID {
				if match.Player2Card != nil {
					return false, "Você já fez sua jogada"
				}
				currentPlayer = match.Player2
			} else {
				return false, "Você não faz parte desta partida"
			}

			// Verifica se o jogador tem a carta no inventário
			if !currentPlayer.HasCardType(cardType) {
				return false, fmt.Sprintf("Você não possui cartas do tipo %s no seu inventário!", cardType)
			}

			// Remove a carta do inventário do jogador
			if !currentPlayer.RemoveCard(cardType) {
				return false, "Erro ao remover carta do inventário"
			}

			// Cria a carta jogada
			playedCard := card.Card{Type: cardType, Rarity: "comum"} // Rarity não importa no jogo

			// Registra a jogada
			if match.Player1.GetID() == playerID {
				match.Player1Card = &playedCard
				fmt.Printf("🃏 Player1 (ID: %d) jogou: %s (removida do inventário)\n", playerID, cardType)
			} else {
				match.Player2Card = &playedCard
				fmt.Printf("🃏 Player2 (ID: %d) jogou: %s (removida do inventário)\n", playerID, cardType)
			}

			// Verifica se ambos jogaram para determinar vencedor
			if match.Player1Card != nil && match.Player2Card != nil {
				return mm.finishCardGame(match)
			} else {
				// Passa o turno para o outro jogador
				if match.CurrentTurn == match.Player1.GetID() {
					match.CurrentTurn = match.Player2.GetID()
				} else {
					match.CurrentTurn = match.Player1.GetID()
				}
				return true, fmt.Sprintf("Carta %s jogada com sucesso! Aguardando o oponente...", cardType)
			}
		}
	}
	return false, "Partida não encontrada"
}


func (mm *MatchManager) finishCardGame(match *Match) (bool, string) {
	player1Card := *match.Player1Card
	player2Card := *match.Player2Card
	
	// Usa a lógica do sistema de cartas para determinar vencedor
	winner, message := card.DetermineWinner(player1Card, player2Card)
	
	var winnerID int
	var finalMessage string
	
	switch winner {
	case 0: // Empate
		// Em caso de empate, Player1 vence (regra da casa)
		winnerID = match.Player1.GetID()
		finalMessage = fmt.Sprintf("Empate! Ambos jogaram %s. %s vence por começar primeiro", 
			player1Card.Type, match.Player1.GetUserName())
	case 1: // Player1 vence
		winnerID = match.Player1.GetID()
		finalMessage = fmt.Sprintf("%s (%s) vs %s (%s): %s", 
			match.Player1.GetUserName(), player1Card.Type,
			match.Player2.GetUserName(), player2Card.Type, message)
	case 2: // Player2 vence
		winnerID = match.Player2.GetID()
		finalMessage = fmt.Sprintf("%s (%s) vs %s (%s): %s", 
			match.Player1.GetUserName(), player1Card.Type,
			match.Player2.GetUserName(), player2Card.Type, message)
	default:
		winnerID = match.Player1.GetID()
		finalMessage = "Erro na determinação do vencedor. Player1 vence por padrão."
	}
	
	match.Status = "finished"
	match.Winner = winnerID
	
	fmt.Printf("🏆 %s\n", finalMessage)
	return true, finalMessage
}


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


func (mm *MatchManager) GetAllActiveMatches() []Match {
	mm.mutex.Lock()
	defer mm.mutex.Unlock()

	var activeMatches []Match
	for _, match := range mm.matches {
		if match.Status == "waiting" || match.Status == "playing" {
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