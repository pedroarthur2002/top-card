package client

import (
	"bufio"
	"net"
	"fmt"
	"os"
	"strings"
	"strconv"
	"time"
	"sync"
	"top-card/internal/protocol"
	"golang.org/x/net/icmp"
	"golang.org/x/net/ipv4"
)

var currentMatchID int
var currentUserID int
var isLoggedIn bool
var inMatch bool // Flag para indicar se está em partida
var isMyTurn bool = false  // Flag para controlar se é o turno do jogador

var isConnected = true
var connectionMutex sync.Mutex

var playerInventory []protocol.CardInfo // Inventário local do jogador

// Canais para comunicação entre goroutines
var syncResponseChan = make(chan []byte, 10)
var asyncMessageChan = make(chan []byte, 10)

func Run() {
	serverAddr := os.Getenv("SERVER_ADDR")
	if serverAddr == "" {
		serverAddr = "localhost:8080"
	}

	conn, err := net.Dial("tcp", serverAddr)
	if err != nil {
		fmt.Println("Erro ao conectar no servidor:", err)
		return
	}
	defer conn.Close()

	fmt.Println("Conectado ao servidor TOP CARD!")

	go messageDistributor(conn)
	go asyncMessageProcessor()

	reader := bufio.NewReader(os.Stdin)
	
	for {
		clearScreen()

		connectionMutex.Lock()
		connected := isConnected
		connectionMutex.Unlock()
		
		fmt.Println("\n========================")
		fmt.Println("Bem vindo ao TOP CARD!")
		fmt.Println("========================")
		
		// Mostra status da conexão
		if !connected {
			fmt.Println("🔴 DESCONECTADO")
			fmt.Println("⚠️  Dados perdidos - registre-se novamente")
		}

		if inMatch {
			fmt.Println("🎮 Você está em uma partida!")
		}
		
		fmt.Println("1 - Fazer login")
		fmt.Println("2 - Cadastrar-se")
		fmt.Println("3 - Abrir pacote de cartas")
		fmt.Println("4 - Buscar partida")
		fmt.Println("5 - Verificar ping")
		fmt.Println("6 - Fazer jogada")        
		fmt.Println("7 - Ver estatísticas")
		if !connected {
			fmt.Println("9 - 🔄 RECONECTAR AO SERVIDOR")  // Destaque quando desconectado
		} else {
			fmt.Println("9 - Reconectar ao servidor")
		}
		fmt.Println("8 - Sair")
		
		fmt.Print("Insira sua opção: ")
		input, _ := reader.ReadString('\n')
		input = strings.TrimSpace(input)             
		choice, _ := strconv.Atoi(input)

		switch choice {
		case 1:
			if isLoggedIn {
				fmt.Println("Você já está logado!")
				continue
			}
			handleLogin(conn, reader)
			
		case 2:
			if isLoggedIn {
				fmt.Println("Você já está logado! Faça logout primeiro.")
				continue
			}
			handleRegister(conn, reader)

		case 3:
			if !isLoggedIn{
				fmt.Println("Você precisa estar logado para abrir os pacotes de cartas!")
				continue
			}
			handleCardPack(conn)

		case 4:
			if !isLoggedIn {
				fmt.Println("Você precisa estar logado para buscar partida!")
				continue
			}
			if inMatch {
				fmt.Println("Você já está em uma partida!")
				continue
			}
			handleQueue(conn)
			
		case 5:
			if !isLoggedIn {
				fmt.Println("Você precisa estar logado para solicitar o ping!")
				continue
			}
			handlePing(conn)

		case 6:
			if !isLoggedIn {
				fmt.Println("Você precisa estar logado para jogar!")
				continue
			}
			if !inMatch {
				fmt.Println("Você precisa estar em uma partida para jogar!")
				continue
			}
			handleGameMove(conn, reader)

		case 7:
			if !isLoggedIn {
				fmt.Println("Você precisa estar logado para ver suas estatísticas!")
				continue
			}
			handleStats(conn)
		
		case 9:
			attemptReconnection(serverAddr, &conn)
			
		case 8:
			fmt.Println("Você escolheu sair. Saindo...")
			return
			
		default:
			fmt.Println("Opção inválida!")
		}
	}
}

// Goroutine que distribui mensagens entre síncronas e assíncronas
func messageDistributor(conn net.Conn) {
	serverReader := bufio.NewScanner(conn)
	
	for serverReader.Scan() {
		responseData := serverReader.Bytes()
		
		dataCopy := make([]byte, len(responseData))
		copy(dataCopy, responseData)
		
		message, err := protocol.DecodeMessage(dataCopy)
		if err != nil {
			fmt.Printf("\n🔴 Erro ao decodificar mensagem do servidor: %v\n", err)
			continue
		}

		switch message.Type {
		case protocol.MSG_LOGIN_RESPONSE, protocol.MSG_REGISTER_RESPONSE, protocol.MSG_QUEUE_RESPONSE, protocol.MSG_PING_RESPONSE, protocol.MSG_STATS_RESPONSE, protocol.MSG_CARD_PACK_RESPONSE:
			select {
			case syncResponseChan <- dataCopy:
			case <-time.After(100 * time.Millisecond):
				fmt.Printf("\n⚠️ Timeout ao enviar resposta síncrona\n")
			}
		case protocol.MSG_MATCH_FOUND, protocol.MSG_MATCH_START, protocol.MSG_MATCH_END, protocol.MSG_GAME_STATE, protocol.MSG_TURN_UPDATE:
			select {
			case asyncMessageChan <- dataCopy:
			case <-time.After(100 * time.Millisecond):
				fmt.Printf("\n⚠️ Timeout ao enviar mensagem assíncrona\n")
			}
		default:
			fmt.Printf("\n⚠️ Tipo de mensagem desconhecido: %s\n", message.Type)
		}
	}

	// NOVA PARTE: Quando sair do loop, conexão foi perdida
	connectionMutex.Lock()
	isConnected = false
	connectionMutex.Unlock()

	clearPlayerData()
	
	fmt.Println("\n🔴 ==========================================")
	fmt.Println("        SERVIDOR DESCONECTADO")
	fmt.Println("🔴 ==========================================")
	fmt.Println("❌ Conexão com o servidor foi perdida")
	fmt.Println("📋 Todos os dados foram perdidos no servidor")
	fmt.Println("🔄 Você precisará se REGISTRAR novamente")
	fmt.Println("💡 Use a opção 9 para reconectar")
	fmt.Println("==========================================")
	
	if err := serverReader.Err(); err != nil {
		fmt.Printf("   Detalhes do erro: %v\n", err)
	}
}

// Goroutine para processar mensagens assíncronas
func asyncMessageProcessor() {
	for {
		select {
		case data := <-asyncMessageChan:
			// Decodifica a mensagem
			message, err := protocol.DecodeMessage(data)
			if err != nil {
				fmt.Printf("\n🔴 Erro ao decodificar mensagem assíncrona: %v\n", err)
				continue
			}

			// Processa mensagens assíncronas
			switch message.Type {
			case protocol.MSG_MATCH_FOUND:
				handleMatchFound(message)
			case protocol.MSG_MATCH_START:
				handleMatchStart(message)
			case protocol.MSG_MATCH_END:
				handleMatchEnd(message)
			case protocol.MSG_GAME_STATE:
				handleGameState(message)
			case protocol.MSG_TURN_UPDATE:
				handleTurnUpdate(message)
			}
		}
	}
}

// Função helper para aguardar resposta síncrona
func waitForSyncResponse(timeout time.Duration) ([]byte, error) {
	select {
	case data := <-syncResponseChan:
		return data, nil
	case <-time.After(timeout):
		connectionMutex.Lock()
		connected := isConnected
		connectionMutex.Unlock()
		
		if !connected {
			return nil, fmt.Errorf("servidor desconectado - todos os dados foram perdidos")
		}
		return nil, fmt.Errorf("timeout - servidor não respondeu")
	}
}

// Função helper para verificar conexão antes de fazer requisições
func checkConnection() bool {
	connectionMutex.Lock()
	connected := isConnected
	connectionMutex.Unlock()
	
	if !connected {
		fmt.Println("❌ Não conectado ao servidor!")
		fmt.Println("💡 Use a opção 9 para reconectar")
		return false
	}
	return true
}

// Manipula notificação de partida encontrada
func handleMatchFound(message *protocol.Message) {
	matchFound, err := protocol.ExtractMatchFound(message)
	if err != nil {
		fmt.Printf("\n🔴 Erro ao extrair dados de partida encontrada: %v\n", err)
		return
	}

	currentMatchID = matchFound.MatchID // Armazena o ID da partida atual

	fmt.Printf("\n\n🎯 ===== PARTIDA ENCONTRADA! =====\n")
	fmt.Printf("🎮 Match ID: %d\n", matchFound.MatchID)
	fmt.Printf("⚔️ Oponente: %s (ID: %d)\n", matchFound.OpponentName, matchFound.OpponentID)
	fmt.Printf("📝 %s\n", matchFound.Message)
	fmt.Printf("⏳ Preparando a partida...\n")
	fmt.Printf("==================================\n")
	// Remove o "Pressione Enter para continuar" para evitar confusão
	
	inMatch = true
}

// Manipula estado do jogo
func handleGameState(message *protocol.Message) {
	gameState, err := protocol.ExtractGameState(message)
	if err != nil {
		fmt.Printf("\n🔴 Erro ao extrair estado do jogo: %v\n", err)
		return
	}

	// NOVA PARTE: Atualiza flag de turno
	isMyTurn = gameState.YourTurn

	fmt.Printf("\n\n🎮 ===== ESTADO DO JOGO =====\n")
	fmt.Printf("📝 %s\n", gameState.Message)
	
	if gameState.YourTurn && !gameState.GameOver {
		fmt.Printf("🎯 É SEU TURNO! Use a opção 6 do menu para jogar.\n")
	} else if !gameState.GameOver {
		fmt.Printf("⏳ Aguardando o oponente jogar...\n")
	}
	
	fmt.Printf("============================\n")
}

// Manipula atualização de turno
func handleTurnUpdate(message *protocol.Message) {
	turnUpdate, err := protocol.ExtractTurnUpdate(message)
	if err != nil {
		fmt.Printf("\n🔴 Erro ao extrair atualização de turno: %v\n", err)
		return
	}

	// NOVA PARTE: Atualiza flag de turno
	isMyTurn = turnUpdate.YourTurn

	fmt.Printf("\n\n🔄 ===== ATUALIZAÇÃO =====\n")
	fmt.Printf("📝 %s\n", turnUpdate.Message)
	
	if turnUpdate.YourTurn {
		fmt.Printf("🎯 É SEU TURNO! Use a opção 6 do menu para jogar.\n")
	}
	
	fmt.Printf("========================\n")
}

// Manipula notificação de início de partida
func handleMatchStart(message *protocol.Message) {
	matchStart, err := protocol.ExtractMatchStart(message)
	if err != nil {
		fmt.Printf("\n🔴 Erro ao extrair dados de início de partida: %v\n", err)
		return
	}

	fmt.Printf("\n\n🚀 ===== PARTIDA INICIADA! =====\n")
	fmt.Printf("🎮 Match ID: %d\n", matchStart.MatchID)
	fmt.Printf("🎯 %s\n", matchStart.Message)
	fmt.Printf("⚔️ Que comece a batalha!\n")
	fmt.Printf("📋 Use a opção 6 do menu quando for seu turno!\n")
	fmt.Printf("===============================\n")
	
	inMatch = true
	// NOVA PARTE: Inicializa o turno como false - será atualizado pelo GameState
	isMyTurn = false
}

// Manipula notificação de fim de partida
func handleMatchEnd(message *protocol.Message) {
	matchEnd, err := protocol.ExtractMatchEnd(message)
	if err != nil {
		fmt.Printf("\n🔴 Erro ao extrair dados de fim de partida: %v\n", err)
		return
	}

	fmt.Printf("\n\n🏆 ===== PARTIDA FINALIZADA! =====\n")
	fmt.Printf("🎮 Match ID: %d\n", matchEnd.MatchID)
	
	if matchEnd.WinnerID == currentUserID {
		fmt.Printf("🎉 VITÓRIA! Você ganhou!\n")
	} else {
		fmt.Printf("😔 DERROTA! Vencedor: %s (ID: %d)\n", matchEnd.WinnerName, matchEnd.WinnerID)
	}
	
	fmt.Printf("📝 %s\n", matchEnd.Message)
	fmt.Printf("🔄 Voltando ao menu principal...\n")
	fmt.Printf("=================================\n")
	
	inMatch = false
	currentMatchID = 0
	// NOVA PARTE: Reseta o turno
	isMyTurn = false
}

func handleGameMove(conn net.Conn, reader *bufio.Reader) {
	if !checkConnection(){
		return
	}

	// NOVA VALIDAÇÃO: Verifica se é o turno do jogador
	if !isMyTurn {
		fmt.Println("❌ Não é seu turno! Aguarde o oponente jogar.")
		fmt.Println("💡 Você será notificado quando for sua vez.")
		return
	}

	// Verifica se tem cartas
	hydra, quimera, gorgona := getCurrentPlayerCards()
	if hydra == 0 && quimera == 0 && gorgona == 0 {
		fmt.Println("❌ Você não tem cartas! Abra um pacote primeiro.")
		return
	}

	fmt.Println("\n--- FAZER JOGADA COM CARTA ---")
	fmt.Printf("📋 Seu inventário: HYDRA(%d) | QUIMERA(%d) | GORGONA(%d)\n", hydra, quimera, gorgona)
	fmt.Println("Escolha uma carta para jogar:")
	
	// Mostra apenas cartas disponíveis
	validChoices := make(map[int]string)
	choiceNum := 1
	
	if hydra > 0 {
		fmt.Printf("%d - HYDRA (devora QUIMERA) - Disponível: %d\n", choiceNum, hydra)
		validChoices[choiceNum] = "HYDRA"
		choiceNum++
	}
	
	if quimera > 0 {
		fmt.Printf("%d - QUIMERA (destrói GORGONA) - Disponível: %d\n", choiceNum, quimera)
		validChoices[choiceNum] = "QUIMERA"
		choiceNum++
	}
	
	if gorgona > 0 {
		fmt.Printf("%d - GORGONA (petrifica HYDRA) - Disponível: %d\n", choiceNum, gorgona)
		validChoices[choiceNum] = "GORGONA"
		choiceNum++
	}
	
	fmt.Printf("Digite sua escolha (1-%d): ", len(validChoices))
	
	input, _ := reader.ReadString('\n')
	input = strings.TrimSpace(input)
	
	choice, err := strconv.Atoi(input)
	if err != nil || choice < 1 || choice > len(validChoices) {
		fmt.Printf("❌ Por favor, digite uma opção válida (1-%d)!\n", len(validChoices))
		return
	}

	cardType, exists := validChoices[choice]
	if !exists {
		fmt.Println("❌ Opção inválida!")
		return
	}

	// Verifica novamente se tem a carta (double check)
	if !hasCardType(cardType) {
		fmt.Printf("❌ Você não possui cartas do tipo %s!\n", cardType)
		return
	}

	// Cria a mensagem de jogada com carta
	cardMoveMessage, err := protocol.CreateCardMove(currentUserID, currentMatchID, cardType)
	if err != nil {
		fmt.Println("Erro ao criar mensagem de jogada:", err)
		return
	}

	// Adiciona quebra de linha
	cardMoveMessage = append(cardMoveMessage, '\n')

	// Envia para o servidor
	_, err = conn.Write(cardMoveMessage)
	if err != nil {
		fmt.Println("Erro ao enviar jogada:", err)
		return
	}

	// Remove a carta do inventário local (otimista - assume que o servidor aceitará)
	removeCardFromLocal(cardType)

	// NOVA PARTE: Marca que não é mais o turno do jogador
	isMyTurn = false

	fmt.Printf("✅ Carta jogada: %s\n", cardType)
	fmt.Println("⏳ Aguardando resposta do servidor...")
}

// função para lidar com pacotes de cartas
func handleCardPack(conn net.Conn) {
	if !checkConnection(){
		return
	}

	// NOVA VALIDAÇÃO: Verifica se já tem cartas
	hydra, quimera, gorgona := getCurrentPlayerCards()
	totalCards := hydra + quimera + gorgona
	
	if totalCards > 0 {
		fmt.Println("\n❌ VOCÊ JÁ POSSUI CARTAS!")
		fmt.Printf("📋 Seu inventário atual: HYDRA(%d) | QUIMERA(%d) | GORGONA(%d)\n", hydra, quimera, gorgona)
		fmt.Println("💡 Use suas cartas em partidas antes de abrir novos pacotes.")
		fmt.Println("\nPressione Enter para continuar...")
		bufio.NewReader(os.Stdin).ReadString('\n')
		return
	}

	fmt.Println("\n--- ABRIR PACOTE DE CARTAS ---")
	fmt.Println("🎒 Abrindo pacote de cartas...")

	// Cria a mensagem de requisição de pacote
	cardPackMessage, err := protocol.CreateCardPackRequest(currentUserID)
	if err != nil {
		fmt.Println("Erro ao criar mensagem de pacote de cartas:", err)
		return
	}

	// Adiciona quebra de linha
	cardPackMessage = append(cardPackMessage, '\n')

	// Envia para o servidor
	_, err = conn.Write(cardPackMessage)
	if err != nil {
		fmt.Println("Erro ao enviar requisição de pacote:", err)
		return
	}

	// Aguarda resposta síncrona
	responseData, err := waitForSyncResponse(5 * time.Second)
	if err != nil {
		fmt.Println("Erro:", err)
		return
	}

	// Decodifica a resposta
	message, err := protocol.DecodeMessage(responseData)
	if err != nil {
		fmt.Println("Erro ao decodificar resposta:", err)
		return
	}

	// Processa resposta de pacote de cartas
	if message.Type == protocol.MSG_CARD_PACK_RESPONSE {
		cardPackResp, err := protocol.ExtractCardPackResponse(message)
		if err != nil {
			fmt.Println("Erro ao extrair resposta de pacote:", err)
			return
		}

		if cardPackResp.Success {
			fmt.Printf("✅ %s\n", cardPackResp.Message)
			
			if len(cardPackResp.Cards) > 0 {
				// Atualiza o inventário local
				updateLocalInventory(cardPackResp.Cards)
				
				fmt.Println("\n🃏 ===== SUAS CARTAS =====")
				for i, card := range cardPackResp.Cards {
					rarityEmoji := ""
					switch card.Rarity {
					case "comum":
						rarityEmoji = "⚪"
					case "raro":
						rarityEmoji = "🔵"
					case "épico":
						rarityEmoji = "🟣"
					}
					fmt.Printf("%d. %s %s (%s)\n", i+1, rarityEmoji, card.Type, card.Rarity)
				}
				fmt.Println("========================")
				
				// Mostra inventário total
				hydra, quimera, gorgona := getCurrentPlayerCards()
				fmt.Printf("\n📋 SEU INVENTÁRIO TOTAL:\n")
				fmt.Printf("⚪ HYDRA: %d cartas\n", hydra)
				fmt.Printf("🔵 QUIMERA: %d cartas\n", quimera)
				fmt.Printf("🟣 GORGONA: %d cartas\n", gorgona)
				fmt.Printf("📊 Total: %d cartas\n", hydra+quimera+gorgona)
			}
			
			// Mostra informações do estoque
			stock := cardPackResp.StockInfo
			fmt.Printf("\n📦 Estoque Global Restante:\n")
			fmt.Printf("⚪ HYDRA: %d cartas\n", stock.HydraCount)
			fmt.Printf("🔵 QUIMERA: %d cartas\n", stock.QuimeraCount)
			fmt.Printf("🟣 GORGONA: %d cartas\n", stock.GorgonaCount)
			fmt.Printf("📊 Total: %d cartas\n", stock.TotalCards)
			
		} else {
			fmt.Printf("❌ %s\n", cardPackResp.Message)
		}
	}

	fmt.Println("\nPressione Enter para continuar...")
	bufio.NewReader(os.Stdin).ReadString('\n')
}

func handleQueue(conn net.Conn) {
	if !checkConnection(){
		return
	}

	fmt.Println("\n--- BUSCAR PARTIDA ---")
	fmt.Println("Entrando na fila de partidas...")

	// Cria a mensagem de requisição de fila
	queueMessage, err := protocol.CreateQueueRequest(currentUserID)
	if err != nil {
		fmt.Println("Erro ao criar mensagem de fila:", err)
		return
	}

	// Adiciona quebra de linha para o servidor conseguir ler
	queueMessage = append(queueMessage, '\n')

	// Envia para o servidor
	_, err = conn.Write(queueMessage)
	if err != nil {
		fmt.Println("Erro ao enviar requisição de fila:", err)
		return
	}

	// Aguarda resposta síncrona
	responseData, err := waitForSyncResponse(5 * time.Second)
	if err != nil {
		fmt.Println("Erro:", err)
		return
	}

	// Decodifica a resposta
	message, err := protocol.DecodeMessage(responseData)
	if err != nil {
		fmt.Println("Erro ao decodificar resposta:", err)
		return
	}

	// Processa resposta de fila
	if message.Type == protocol.MSG_QUEUE_RESPONSE {
		queueResp, err := protocol.ExtractQueueResponse(message)
		if err != nil {
			fmt.Println("Erro ao extrair resposta de fila:", err)
			return
		}

		if queueResp.Success {
			fmt.Printf("✅ %s\n", queueResp.Message)
			fmt.Printf("Jogadores na fila: %d\n", queueResp.QueueSize)
			fmt.Println("🔍 Aguardando por oponentes...")
			fmt.Println("💡 As notificações de partida aparecerão automaticamente!")
		} else {
			fmt.Printf("❌ %s\n", queueResp.Message)
		}
	}
}

func handleRegister(conn net.Conn, reader *bufio.Reader) {
	if !checkConnection(){
		return
	}

	fmt.Println("\n--- CADASTRO ---")
	fmt.Print("Insira um nome de usuário (mín. 3 caracteres): ")
	userName, _ := reader.ReadString('\n')
	userName = strings.TrimSpace(userName)
	
	fmt.Print("Digite sua senha (mín. 4 caracteres): ")
	password, _ := reader.ReadString('\n')
	password = strings.TrimSpace(password)

	// Validações básicas no cliente
	if len(userName) < 3 {
		fmt.Println("❌ Nome de usuário deve ter pelo menos 3 caracteres!")
		return
	}
	
	if len(password) < 4 {
		fmt.Println("❌ Senha deve ter pelo menos 4 caracteres!")
		return
	}

	// Cria a mensagem de cadastro
	registerMessage, err := protocol.CreateRegisterRequest(userName, password)
	if err != nil {
		fmt.Println("Erro ao criar mensagem de cadastro:", err)
		return
	}

	// Adiciona quebra de linha para o servidor conseguir ler
	registerMessage = append(registerMessage, '\n')

	// Envia para o servidor
	_, err = conn.Write(registerMessage)
	if err != nil {
		fmt.Println("Erro ao enviar cadastro:", err)
		return
	}

	// Aguarda resposta síncrona
	responseData, err := waitForSyncResponse(5 * time.Second)
	if err != nil {
		fmt.Println("Erro:", err)
		return
	}

	// Decodifica a resposta
	message, err := protocol.DecodeMessage(responseData)
	if err != nil {
		fmt.Println("Erro ao decodificar resposta:", err)
		return
	}

	// Processa resposta de cadastro
	if message.Type == protocol.MSG_REGISTER_RESPONSE {
		registerResp, err := protocol.ExtractRegisterResponse(message)
		if err != nil {
			fmt.Println("Erro ao extrair resposta de cadastro:", err)
			return
		}

		if registerResp.Success {
			fmt.Printf("✅ %s\n", registerResp.Message)
			fmt.Printf("Seu ID de usuário é: %d\n", registerResp.UserID)
			fmt.Println("Agora você pode fazer login!")
		} else {
			fmt.Printf("❌ %s\n", registerResp.Message)
		}
	}
}

func handleLogin(conn net.Conn, reader *bufio.Reader) {
	if !checkConnection(){
		return
	}

	fmt.Println("\n--- LOGIN ---")
	fmt.Print("Insira seu nome de usuário: ")
	userName, _ := reader.ReadString('\n')
	userName = strings.TrimSpace(userName)
	
	fmt.Print("Digite sua senha: ")
	password, _ := reader.ReadString('\n')
	password = strings.TrimSpace(password)

	// Cria a mensagem de login
	loginMessage, err := protocol.CreateLoginRequest(userName, password)
	if err != nil {
		fmt.Println("Erro ao criar mensagem de login:", err)
		return
	}

	// Adiciona quebra de linha para o servidor conseguir ler
	loginMessage = append(loginMessage, '\n')

	// Envia para o servidor
	_, err = conn.Write(loginMessage)
	if err != nil {
		fmt.Println("Erro ao enviar login:", err)
		return
	}

	// Aguarda resposta síncrona
	responseData, err := waitForSyncResponse(5 * time.Second)
	if err != nil {
		fmt.Println("Erro:", err)
		return
	}

	// Decodifica a resposta
	message, err := protocol.DecodeMessage(responseData)
	if err != nil {
		fmt.Println("Erro ao decodificar resposta:", err)
		return
	}

	// Processa resposta de login
	if message.Type == protocol.MSG_LOGIN_RESPONSE {
		loginResp, err := protocol.ExtractLoginResponse(message)
		if err != nil {
			fmt.Println("Erro ao extrair resposta de login:", err)
			return
		}

		if loginResp.Success {
			fmt.Printf("✅ %s\n", loginResp.Message)
			currentUserID = loginResp.UserID
			isLoggedIn = true
			fmt.Printf("Você está logado com ID: %d\n", currentUserID)
		} else {
			fmt.Printf("❌ %s\n", loginResp.Message)
		}
	}
}

func handleStats(conn net.Conn) {
	if !checkConnection(){
		return
	}

	fmt.Println("\n--- SUAS ESTATÍSTICAS ---")

	// Cria a mensagem de requisição de estatísticas
	statsMessage, err := protocol.CreateStatsRequest(currentUserID)
	if err != nil {
		fmt.Println("Erro ao criar mensagem de estatísticas:", err)
		return
	}

	// Adiciona quebra de linha
	statsMessage = append(statsMessage, '\n')

	// Envia para o servidor
	_, err = conn.Write(statsMessage)
	if err != nil {
		fmt.Println("Erro ao enviar requisição de estatísticas:", err)
		return
	}

	// Aguarda resposta síncrona
	responseData, err := waitForSyncResponse(5 * time.Second)
	if err != nil {
		fmt.Println("Erro:", err)
		return
	}

	// Decodifica a resposta
	message, err := protocol.DecodeMessage(responseData)
	if err != nil {
		fmt.Println("Erro ao decodificar resposta:", err)
		return
	}

	// Processa resposta de estatísticas
	if message.Type == protocol.MSG_STATS_RESPONSE {
		statsResp, err := protocol.ExtractStatsResponse(message)
		if err != nil {
			fmt.Println("Erro ao extrair resposta de estatísticas:", err)
			return
		}

		if statsResp.Success {
			fmt.Printf("\n📊 ===== ESTATÍSTICAS DE %s =====\n", statsResp.UserName)
			fmt.Printf("🏆 Vitórias: %d\n", statsResp.Wins)
			fmt.Printf("😔 Derrotas: %d\n", statsResp.Losses)
			fmt.Printf("🎯 Taxa de vitória: %.1f%%\n", statsResp.WinRate)
			
			totalGames := statsResp.Wins + statsResp.Losses
			fmt.Printf("🎮 Total de partidas: %d\n", totalGames)
			
			if totalGames == 0 {
				fmt.Printf("💡 Dica: Jogue algumas partidas para ver suas estatísticas!\n")
			}
			fmt.Printf("========================================\n")
		} else {
			fmt.Printf("❌ %s\n", statsResp.Message)
		}
	}
}

// função de ping
func handlePing(conn net.Conn) {
	if !checkConnection(){
		return
	}

	if !isLoggedIn {
		fmt.Println("❌ Você precisa estar logado para verificar o ping!")
		return
	}
	
	if currentUserID <= 0 {
		fmt.Println("❌ ID de usuário inválido!")
		return
	}

	fmt.Println("\n--- CONSULTA DE PING ICMP ---")
	fmt.Println("🏓 Verificando latência via ICMP...")

	// Obtém endereço do servidor da conexão TCP existente
	serverAddr := conn.RemoteAddr().(*net.TCPAddr).IP.String()
	
	// Realiza ping ICMP
	latencia, err := realizarPingICMP(serverAddr)
	if err != nil {
		fmt.Printf("❌ Erro ao realizar ping ICMP: %v\n", err)
		fmt.Println("💡 Dica: Execute como administrador/root para usar ICMP")
		fmt.Println("🔄 Tentando fallback para ping TCP...")
		
		// Fallback para o ping TCP original se ICMP falhar
		realizarPingTCPFallback(conn)
		return
	}

	fmt.Printf("✅ Ping ICMP realizado com sucesso!\n")
	fmt.Printf("🏓 Latência: %d ms\n", latencia)
	fmt.Printf("📡 Destino: %s\n", serverAddr)
}

// função para realizar ping ICMP real
func realizarPingICMP(endereco string) (int64, error) {
	// Resolve o endereço
	destino, err := net.ResolveIPAddr("ip4", endereco)
	if err != nil {
		return 0, fmt.Errorf("erro ao resolver endereço: %v", err)
	}

	// Cria conexão ICMP
	conexao, err := icmp.ListenPacket("ip4:icmp", "0.0.0.0")
	if err != nil {
		return 0, fmt.Errorf("erro ao criar socket ICMP: %v", err)
	}
	defer conexao.Close()

	// Cria mensagem ICMP Echo Request
	mensagem := &icmp.Message{
		Type: ipv4.ICMPTypeEcho,
		Code: 0,
		Body: &icmp.Echo{
			ID:   os.Getpid() & 0xffff,
			Seq:  1,
			Data: []byte("TOP CARD PING"),
		},
	}

	dados, err := mensagem.Marshal(nil)
	if err != nil {
		return 0, fmt.Errorf("erro ao serializar mensagem ICMP: %v", err)
	}

	// Registra tempo de início
	inicio := time.Now()

	// Envia ping
	_, err = conexao.WriteTo(dados, destino)
	if err != nil {
		return 0, fmt.Errorf("erro ao enviar ping: %v", err)
	}

	// Define timeout para leitura
	err = conexao.SetReadDeadline(time.Now().Add(3 * time.Second))
	if err != nil {
		return 0, fmt.Errorf("erro ao definir timeout: %v", err)
	}

	// Lê resposta
	resposta := make([]byte, 1500)
	n, peer, err := conexao.ReadFrom(resposta)
	if err != nil {
		return 0, fmt.Errorf("erro ao receber resposta: %v", err)
	}

	// Calcula duração
	duracao := time.Since(inicio)

	// Analisa resposta (protocolo IPv4 = 1)
	const protocoloIPv4 = 1
	mensagemResposta, err := icmp.ParseMessage(protocoloIPv4, resposta[:n])
	if err != nil {
		return 0, fmt.Errorf("erro ao analisar resposta: %v", err)
	}

	// Verifica se é Echo Reply (tipo 0)
	if mensagemResposta.Type != ipv4.ICMPTypeEchoReply {
		return 0, fmt.Errorf("tipo de resposta inesperado: %v (esperado: %v)", 
			mensagemResposta.Type, ipv4.ICMPTypeEchoReply)
	}

	fmt.Printf("📡 Resposta de: %v\n", peer)
	
	return duracao.Milliseconds(), nil
}

func realizarPingTCPFallback(conn net.Conn) {
	fmt.Println("\n--- FALLBACK: PING TCP ---")
	fmt.Println("🏓 Verificando latência via TCP...")

	// Registra o tempo de envio
	tempoInicio := time.Now()

	// Cria a mensagem de requisição do ping
	mensagemPing, err := protocol.CreatePingRequest(currentUserID)
	if err != nil {
		fmt.Println("Erro ao solicitar o ping:", err)
		return
	}

	// Adiciona quebra de linha
	mensagemPing = append(mensagemPing, '\n')

	// Envia para o servidor
	_, err = conn.Write(mensagemPing)
	if err != nil {
		fmt.Println("Erro ao enviar requisição de ping:", err)
		return
	}

	// Aguarda resposta síncrona
	dadosResposta, err := waitForSyncResponse(5 * time.Second)
	if err != nil {
		fmt.Println("Erro:", err)
		return
	}

	// Registra o tempo de recebimento
	tempoFim := time.Now()

	// Decodifica a resposta
	mensagem, err := protocol.DecodeMessage(dadosResposta)
	if err != nil {
		fmt.Println("Erro ao decodificar resposta:", err)
		return
	}

	if mensagem.Type == protocol.MSG_PING_RESPONSE {
		respostaPing, err := protocol.ExtractPingResponse(mensagem)
		if err != nil {
			fmt.Println("Erro ao extrair resposta do ping:", err)
			return
		}

		if respostaPing.Success {
			// Calcula a latência
			latencia := tempoFim.Sub(tempoInicio).Milliseconds()
			
			fmt.Printf("✅ %s\n", respostaPing.Message)
			fmt.Printf("🏓 Latência TCP (round-trip): %d ms\n", latencia)
		} else {
			fmt.Printf("❌ %s\n", respostaPing.Message)
		}
	} else {
		fmt.Printf("⚠️  Tipo de resposta inesperado: %s\n", mensagem.Type)
	}
}

// Função de reconexão
func attemptReconnection(serverAddr string, conn *net.Conn) {
	fmt.Println("\n🔄 =============================")
	fmt.Println("     TENTANDO RECONECTAR...")
	fmt.Println("===============================")
	
	// Fecha conexão anterior se ainda existe
	if *conn != nil {
		(*conn).Close()
	}
	
	newConn, err := net.Dial("tcp", serverAddr)
	if err != nil {
		fmt.Printf("❌ Falha na reconexão: %v\n", err)
		fmt.Println("💡 Verifique se o servidor está rodando")
		return
	}

	*conn = newConn
	
	connectionMutex.Lock()
	isConnected = true
	connectionMutex.Unlock()
	
	fmt.Println("✅ Reconectado com sucesso!")
	fmt.Println("📋 IMPORTANTE: Todos os dados foram perdidos")
	fmt.Println("🔄 Você precisa se REGISTRAR novamente")
	fmt.Println("===============================")
	
	// Reinicia goroutines
	go messageDistributor(*conn)
	go asyncMessageProcessor()
}

func updateLocalInventory(cards []protocol.CardInfo) {
	playerInventory = append(playerInventory, cards...)
}

// Função para verificar cartas do jogador
func getCurrentPlayerCards() (int, int, int) {
	hydraCount := 0
	quimeraCount := 0
	gorgonaCount := 0
	
	for _, card := range playerInventory {
		switch card.Type {
		case "HYDRA":
			hydraCount++
		case "QUIMERA":
			quimeraCount++
		case "GORGONA":
			gorgonaCount++
		}
	}
	
	return hydraCount, quimeraCount, gorgonaCount
}

func hasCardType(cardType string) bool {
	for _, card := range playerInventory {
		if card.Type == cardType {
			return true
		}
	}
	return false
}

// Função para remover carta do inventário local
func removeCardFromLocal(cardType string) {
	for i, card := range playerInventory {
		if card.Type == cardType {
			playerInventory = append(playerInventory[:i], playerInventory[i+1:]...)
			break
		}
	}
}

func clearPlayerData() {
	playerInventory = nil
	isLoggedIn = false
	inMatch = false
	currentMatchID = 0
	currentUserID = 0
	isMyTurn = false
}

// Limpar terminal
func clearScreen() {
	fmt.Print("\033[2J\033[H")
}