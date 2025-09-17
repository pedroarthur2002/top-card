package card

import (
    "math/rand"
    "sync"
    "time"
)

// Tipos de cartas
const (
    HYDRA   = "HYDRA"
    QUIMERA = "QUIMERA"
    GORGONA = "GORGONA"
)

// Estrutura para representar uma carta
type Card struct {
    Type   string `json:"type"`
    Rarity string `json:"rarity"` // "comum", "raro", "épico"
}

// Estrutura para o estoque global de cartas
type CardStock struct {
    hydraCount   int
    quimeraCount int
    gorgonaCount int
    mutex        sync.Mutex
}

// Instância global do estoque
var globalStock = &CardStock{
    hydraCount:   10000,   // Mais comum
    quimeraCount: 7000,    // Mediano
    gorgonaCount: 3000,    // Mais raro
}

// Inicializa o gerador de números aleatórios
func init() {
    rand.Seed(time.Now().UnixNano())
}

// Função para obter cartas do estoque (com mutex para thread safety)
func OpenCardPack() ([]Card, bool) {
    globalStock.mutex.Lock()
    defer globalStock.mutex.Unlock()
    // Verifica se há cartas suficientes no estoque
    totalCards := globalStock.hydraCount + globalStock.quimeraCount + globalStock.gorgonaCount
    if totalCards < 3 {
        return nil, false // Estoque insuficiente
    }
    var pack []Card
   
    // Sorteia 3 cartas
    for i := 0; i < 3; i++ {
        card := drawRandomCard()
        if card.Type == "" {
            // Se não conseguiu sortear carta (estoque vazio), reverte as cartas já retiradas
            for _, revertCard := range pack {
                addCardBackToStock(revertCard)
            }
            return nil, false
        }
        pack = append(pack, card)
    }
    return pack, true
}

// Função interna para sortear uma carta aleatória
func drawRandomCard() Card {
    totalCards := globalStock.hydraCount + globalStock.quimeraCount + globalStock.gorgonaCount
    if totalCards == 0 {
        return Card{} // Estoque vazio
    }
    // Sorteia um número baseado no estoque total
    randomNum := rand.Intn(totalCards)
    if randomNum < globalStock.hydraCount {
        globalStock.hydraCount--
        return Card{Type: HYDRA, Rarity: "comum"}
    } else if randomNum < globalStock.hydraCount + globalStock.quimeraCount {
        globalStock.quimeraCount--
        return Card{Type: QUIMERA, Rarity: "raro"}
    } else {
        globalStock.gorgonaCount--
        return Card{Type: GORGONA, Rarity: "épico"}
    }
}

// Função para adicionar carta de volta ao estoque (para casos de erro)
func addCardBackToStock(card Card) {
    switch card.Type {
    case HYDRA:
        globalStock.hydraCount++
    case QUIMERA:
        globalStock.quimeraCount++
    case GORGONA:
        globalStock.gorgonaCount++
    }
}

// Função para verificar o estoque atual
func GetStockInfo() (int, int, int, int) {
    globalStock.mutex.Lock()
    defer globalStock.mutex.Unlock()
   
    total := globalStock.hydraCount + globalStock.quimeraCount + globalStock.gorgonaCount
    return globalStock.hydraCount, globalStock.quimeraCount, globalStock.gorgonaCount, total
}

// Função para determinar o vencedor entre duas cartas (pedra, papel, tesoura)
func DetermineWinner(card1, card2 Card) (winner int, message string) {
    // HYDRA vence QUIMERA (como pedra vence tesoura)
    // QUIMERA vence GORGONA (como tesoura vence papel)  
    // GORGONA vence HYDRA (como papel vence pedra)
   
    if card1.Type == card2.Type {
        return 0, "Empate! Ambos jogaram " + card1.Type
    }
   
    // Casos onde Jogador 1 vence
    switch {
    case card1.Type == HYDRA && card2.Type == QUIMERA:
        return 1, "HYDRA devora QUIMERA! Jogador 1 vence!"
    case card1.Type == QUIMERA && card2.Type == GORGONA:
        return 1, "QUIMERA destrói GORGONA! Jogador 1 vence!"
    case card1.Type == GORGONA && card2.Type == HYDRA:
        return 1, "GORGONA petrifica HYDRA! Jogador 1 vence!"
    }
    
    // Casos onde Jogador 2 vence (inversos dos casos acima)
    switch {
    case card2.Type == HYDRA && card1.Type == QUIMERA:
        return 2, "HYDRA devora QUIMERA! Jogador 2 vence!"
    case card2.Type == QUIMERA && card1.Type == GORGONA:
        return 2, "QUIMERA destrói GORGONA! Jogador 2 vence!"
    case card2.Type == GORGONA && card1.Type == HYDRA:
        return 2, "GORGONA petrifica HYDRA! Jogador 2 vence!"
    }
    
    // Este caso nunca deveria acontecer se as validações estão corretas
    return 0, "Erro: combinação de cartas não reconhecida"
}