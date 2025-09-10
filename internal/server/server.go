package server

import (
	"bufio"
	"fmt"
	"net"
	"strings"
	"sync"
	"time"
	"top-card/internal/player"
	"top-card/internal/protocol"
	"top-card/internal/match"
	"top-card/internal/card"
)

var players []player.Player
var nextID = 1
var queue []int // Fila de jogadores esperando partida
var queueMutex sync.Mutex
var connectedUsers = make(map[int]bool) // Mapa para rastrear usuários conectados por ID
var connectedMutex sync.Mutex           // Mutex para proteger acesso concurrent ao mapa
var userConnections = make(map[int]net.Conn) // Mapa para armazenar conexões dos usuários
var connectionsMutex sync.Mutex              // Mutex para proteger acesso às conexões

func Run() {
	// Criação do servidor (ouvindo na porta 8080)
	ln, err := net.Listen("tcp", ":8080")
	if err != nil {
		fmt.Println("Erro do tipo: ", err)
		return
	}
	defer ln.Close()

	fmt.Println("Servidor TOP CARD ouvindo na porta 8080...")

	// Inicia o matchmaker em uma goroutine separada
	go matchmaker()

	go cleanupOrphanedMatches()

	for {
		conn, err := ln.Accept()
		if err != nil {
			fmt.Println("Erro do tipo: ", err)
			continue
		}

		fmt.Println("Cliente conectado")
		go handleConnection(conn)
	}
}

// Matchmaker - processa a fila e cria partidas quando há 2 jogadores
func matchmaker() {
	ticker := time.NewTicker(1 * time.Second) // Verifica a cada segundo
	defer ticker.Stop()

	for range ticker.C {
		queueMutex.Lock()
		if len(queue) >= 2 {
			// Pega os dois primeiros jogadores da fila
			player1ID := queue[0]
			player2ID := queue[1]
			
			// Remove eles da fila
			queue = queue[2:]
			
			fmt.Printf("🎯 Matchmaker: Criando partida entre %d e %d\n", player1ID, player2ID)
			
			// Busca os objetos Player
			player1, found1 := findPlayerByID(player1ID)
			player2, found2 := findPlayerByID(player2ID)
			
			if found1 && found2 {
				// Cria a partida
				newMatch := match.GetManager().CreateMatch(player1, player2)
				
				// Notifica os jogadores sobre a partida encontrada
				go notifyMatchFound(player1ID, player2ID, player2.GetUserName(), newMatch.ID)
				go notifyMatchFound(player2ID, player1ID, player1.GetUserName(), newMatch.ID)
				
				// Aguarda um pouco e inicia a partida
				time.Sleep(2 * time.Second)
				match.GetManager().StartMatch(newMatch.ID)
				
				// Notifica o início da partida
				go notifyMatchStart(player1ID, player2ID, newMatch.ID)
			}
		}
		queueMutex.Unlock()
	}
}

// Notifica jogador sobre partida encontrada
func notifyMatchFound(playerID, opponentID int, opponentName string, matchID int) {
	connectionsMutex.Lock()
	conn, exists := userConnections[playerID]
	connectionsMutex.Unlock()
	
	if !exists {
		fmt.Printf("⚠️  Conexão não encontrada para jogador %d\n", playerID)
		return
	}

	message := fmt.Sprintf("Partida encontrada! Você vai enfrentar %s", opponentName)
	response, err := protocol.CreateMatchFound(matchID, opponentID, opponentName, message)
	if err != nil {
		fmt.Printf("Erro ao criar mensagem de partida encontrada: %v\n", err)
		return
	}

	response = append(response, '\n')
	_, err = conn.Write(response)
	if err != nil {
		fmt.Printf("Erro ao enviar notificação de partida encontrada para %d: %v\n", playerID, err)
	}
}

// Notifica jogadores sobre início da partida
func notifyMatchStart(player1ID, player2ID, matchID int) {
	message := "A partida começou! Boa sorte!"
	
	// Inicia o jogo
	time.Sleep(1 * time.Second)
	match.GetManager().StartGame(matchID)
	
	// Notifica jogador 1 (é o primeiro a jogar)
	connectionsMutex.Lock()
	conn1, exists1 := userConnections[player1ID]
	connectionsMutex.Unlock()
	
	if exists1 {
		response, err := protocol.CreateMatchStart(matchID, message)
		if err == nil {
			response = append(response, '\n')
			conn1.Write(response)
		}
		
		// Envia estado do jogo indicando que é o turno do Player1
		gameState, err := protocol.CreateGameState(matchID, "É seu turno! Escolha uma carta para jogar.", true, false, false)
		if err == nil {
			gameState = append(gameState, '\n')
			conn1.Write(gameState)
		}
	}

	// Notifica jogador 2
	connectionsMutex.Lock()
	conn2, exists2 := userConnections[player2ID]
	connectionsMutex.Unlock()
	
	if exists2 {
		response, err := protocol.CreateMatchStart(matchID, message)
		if err == nil {
			response = append(response, '\n')
			conn2.Write(response)
		}
		
		// Envia estado do jogo indicando que deve aguardar
		gameState, err := protocol.CreateGameState(matchID, "Aguardando o oponente escolher uma carta...", false, false, false)
		if err == nil {
			gameState = append(gameState, '\n')
			conn2.Write(gameState)
		}
	}
}


func handleConnection(conn net.Conn) {
	defer func() {
		conn.Close()
		
		// Remove o usuário das estruturas quando desconectar
		connectionsMutex.Lock()
		for userID, userConn := range userConnections {
			if userConn == conn {
				delete(userConnections, userID)
				
				connectedMutex.Lock()
				delete(connectedUsers, userID)
				connectedMutex.Unlock()
				
				// Remove da fila se estiver lá
				queueMutex.Lock()
				for i, playerID := range queue {
					if playerID == userID {
						queue = append(queue[:i], queue[i+1:]...)
						break
					}
				}
				queueMutex.Unlock()
				
				fmt.Printf("👋 Usuário %d desconectado\n", userID)
				break
			}
		}
		connectionsMutex.Unlock()
	}()
	
	scanner := bufio.NewScanner(conn)

	for scanner.Scan() {
		data := scanner.Bytes()
		
		// Decodifica a mensagem recebida
		message, err := protocol.DecodeMessage(data)
		if err != nil {
			fmt.Println("Erro ao decodificar mensagem:", err)
			continue
		}

		// Processa baseado no tipo da mensagem
		switch message.Type {
		case protocol.MSG_LOGIN_REQUEST:
			handleLogin(conn, message)
		case protocol.MSG_REGISTER_REQUEST:
			handleRegister(conn, message)
		case protocol.MSG_QUEUE_REQUEST:
			handleQueue(conn, message)
		case protocol.MSG_PING_REQUEST:
			handlePing(conn, message)
		case protocol.MSG_STATS_REQUEST:  
			handleStats(conn, message)
		case protocol.MSG_CARD_PACK_REQUEST:
			handleCardPack(conn, message)
		case protocol.MSG_CARD_MOVE:
			handleCardMove(conn, message)
		default:
			fmt.Println("Tipo de mensagem não reconhecido:", message.Type)
		}
	}

	if err := scanner.Err(); err != nil {
		fmt.Println("Erro ao ler do cliente:", err)
	}
}


func notifyMatchEnd(player1ID, player2ID int, currentMatch *match.Match) {
	var winnerName string
	if currentMatch.Player1.GetID() == currentMatch.Winner {
		winnerName = currentMatch.Player1.GetUserName()
	} else {
		winnerName = currentMatch.Player2.GetUserName()
	}
	
	// Obtém as cartas jogadas (agora são cartas, não números)
	var player1Card, player2Card string
	if currentMatch.Player1Card != nil {
		player1Card = currentMatch.Player1Card.Type
	} else {
		player1Card = "NENHUMA"
	}
	
	if currentMatch.Player2Card != nil {
		player2Card = currentMatch.Player2Card.Type
	} else {
		player2Card = "NENHUMA"
	}
	
	message := fmt.Sprintf("Jogo finalizado! %s jogou %s vs %s jogou %s", 
		currentMatch.Player1.GetUserName(), player1Card,
		currentMatch.Player2.GetUserName(), player2Card)
	
	// Atualiza as estatísticas dos jogadores
	updatePlayerStats(currentMatch.Player1.GetID(), currentMatch.Player2.GetID(), currentMatch.Winner)
	
	// Notifica jogador 1
	connectionsMutex.Lock()
	conn1, exists1 := userConnections[player1ID]
	connectionsMutex.Unlock()
	
	if exists1 {
		response, err := protocol.CreateMatchEnd(currentMatch.ID, currentMatch.Winner, winnerName, message)
		if err == nil {
			response = append(response, '\n')
			conn1.Write(response)
		}
	}

	// Notifica jogador 2
	connectionsMutex.Lock()
	conn2, exists2 := userConnections[player2ID]
	connectionsMutex.Unlock()
	
	if exists2 {
		response, err := protocol.CreateMatchEnd(currentMatch.ID, currentMatch.Winner, winnerName, message)
		if err == nil {
			response = append(response, '\n')
			conn2.Write(response)
		}
	}
}


func updatePlayerStats(player1ID, player2ID, winnerID int) {
	// Encontra os players no slice e atualiza suas estatísticas
	for i := range players {
		if players[i].GetID() == player1ID {
			if winnerID == player1ID {
				players[i].AddWin()
			} else {
				players[i].AddLoss()
			}
			fmt.Printf("📊 Estatísticas atualizadas para %s: %dW-%dL\n", 
				players[i].GetUserName(), players[i].GetWins(), players[i].GetLosses())
		} else if players[i].GetID() == player2ID {
			if winnerID == player2ID {
				players[i].AddWin()
			} else {
				players[i].AddLoss()
			}
			fmt.Printf("📊 Estatísticas atualizadas para %s: %dW-%dL\n", 
				players[i].GetUserName(), players[i].GetWins(), players[i].GetLosses())
		}
	}
}


// função para lidar com jogadas no servidor
func handleCardMove(conn net.Conn, message *protocol.Message) {
	// Extrai os dados da jogada com carta
	cardMove, err := protocol.ExtractCardMove(message)
	if err != nil {
		fmt.Println("Erro ao extrair dados da jogada de carta:", err)
		return
	}

	fmt.Printf("Jogada de carta recebida - Usuário: %d, Partida: %d, Carta: %s\n", 
		cardMove.UserID, cardMove.MatchID, cardMove.CardType)

	// Processa a jogada de carta
	success, responseMessage := match.GetManager().MakeCardMove(cardMove.MatchID, cardMove.UserID, cardMove.CardType)
	
	if !success {
		// Envia mensagem de erro para o jogador
		response, err := protocol.CreateGameState(cardMove.MatchID, responseMessage, false, false, false)
		if err != nil {
			fmt.Printf("Erro ao criar resposta de erro: %v\n", err)
			return
		}
		
		response = append(response, '\n')
		conn.Write(response)
		return
	}

	// Se a jogada foi bem-sucedida, verifica o estado da partida
	currentMatch := match.GetManager().GetMatch(cardMove.MatchID)
	if currentMatch == nil {
		fmt.Printf("Partida %d não encontrada\n", cardMove.MatchID)
		return
	}

	// Se o jogo terminou (ambos jogaram)
	if currentMatch.Status == "finished" {
		// Notifica fim de partida para ambos jogadores
		go notifyMatchEnd(currentMatch.Player1.GetID(), currentMatch.Player2.GetID(), currentMatch)
	} else {
		// Notifica atualização de turno para ambos jogadores
		go notifyTurnUpdate(currentMatch)
	}
}

func handleCardPack(conn net.Conn, message *protocol.Message) {
	// Extrai os dados da requisição de pacote
	cardPackReq, err := protocol.ExtractCardPackRequest(message)
	if err != nil {
		fmt.Println("Erro ao extrair dados de pacote de cartas:", err)
		return
	}

	fmt.Printf("Requisição de pacote de cartas - UserID: %d\n", cardPackReq.UserID)

	// Verifica se o usuário está conectado
	connectedMutex.Lock()
	isConnected := connectedUsers[cardPackReq.UserID]
	connectedMutex.Unlock()

	var response []byte

	if !isConnected {
		// Usuário não está conectado/autenticado
		response, err = protocol.CreateCardPackResponse(false, "Usuário não está conectado!", nil, protocol.StockInfo{})
		fmt.Printf("Pacote de cartas negado - usuário %d não está conectado\n", cardPackReq.UserID)
	} else {
		// Busca o player
		foundPlayer, found := findPlayerByID(cardPackReq.UserID)
		if !found {
			response, err = protocol.CreateCardPackResponse(false, "Usuário não encontrado!", nil, protocol.StockInfo{})
			fmt.Printf("Pacote de cartas negado - usuário %d não encontrado\n", cardPackReq.UserID)
		} else {
			// NOVA VALIDAÇÃO: Verifica se o jogador já tem cartas
			if foundPlayer.GetInventorySize() > 0 {
				hydra, quimera, gorgona := foundPlayer.CountCardsByType()
				message := fmt.Sprintf("Você já possui %d cartas! Use-as em partidas antes de abrir novos pacotes.", foundPlayer.GetInventorySize())
				response, err = protocol.CreateCardPackResponse(false, message, nil, protocol.StockInfo{})
				fmt.Printf("Pacote de cartas negado - usuário %d já possui cartas (H:%d Q:%d G:%d)\n", 
					cardPackReq.UserID, hydra, quimera, gorgona)
			} else {
				// Tenta abrir um pacote de cartas
				cards, success := card.OpenCardPack()
				if !success {
					response, err = protocol.CreateCardPackResponse(false, "Estoque insuficiente! Tente novamente mais tarde.", nil, protocol.StockInfo{})
					fmt.Printf("Pacote de cartas negado - estoque insuficiente para usuário %d\n", cardPackReq.UserID)
				} else {
					// Adiciona as cartas ao inventário do jogador
					for i := range players {
						if players[i].GetID() == cardPackReq.UserID {
							players[i].AddCards(cards)
							break
						}
					}

					// Converte cartas para protocol.CardInfo
					var cardInfos []protocol.CardInfo
					for _, c := range cards {
						cardInfos = append(cardInfos, protocol.CardInfo{
							Type:   c.Type,
							Rarity: c.Rarity,
						})
					}

					// Obtém informações do estoque
					hydra, quimera, gorgona, total := card.GetStockInfo()
					stockInfo := protocol.StockInfo{
						HydraCount:   hydra,
						QuimeraCount: quimera,
						GorgonaCount: gorgona,
						TotalCards:   total,
					}

					message := fmt.Sprintf("Pacote aberto com sucesso! Você recebeu %d cartas. Agora você deve usá-las antes de abrir outro pacote.", len(cards))
					response, err = protocol.CreateCardPackResponse(true, message, cardInfos, stockInfo)
					
					fmt.Printf("Pacote de cartas aberto para usuário %d: %v (inventário: %d cartas)\n", 
						cardPackReq.UserID, cardInfos, foundPlayer.GetInventorySize()+len(cards))
				}
			}
		}
	}

	if err != nil {
		fmt.Println("Erro ao criar resposta de pacote de cartas:", err)
		return
	}

	// Envia a resposta
	response = append(response, '\n')
	_, err = conn.Write(response)
	if err != nil {
		fmt.Println("Erro ao enviar resposta de pacote de cartas:", err)
	}
}

// Função para notificar atualização de turno
func notifyTurnUpdate(currentMatch *match.Match) {
	player1ID := currentMatch.Player1.GetID()
	player2ID := currentMatch.Player2.GetID()
	
	// Mensagem para Player1
	connectionsMutex.Lock()
	conn1, exists1 := userConnections[player1ID]
	connectionsMutex.Unlock()
	
	if exists1 {
		isPlayer1Turn := currentMatch.CurrentTurn == player1ID
		var message string
		if isPlayer1Turn {
			message = "É seu turno! Escolha uma carta para jogar."
		} else {
			message = "Carta jogada! Aguardando oponente..."
		}
		
		response, err := protocol.CreateTurnUpdate(currentMatch.ID, message, isPlayer1Turn)
		if err == nil {
			response = append(response, '\n')
			conn1.Write(response)
		}
	}

	// Mensagem para Player2
	connectionsMutex.Lock()
	conn2, exists2 := userConnections[player2ID]
	connectionsMutex.Unlock()
	
	if exists2 {
		isPlayer2Turn := currentMatch.CurrentTurn == player2ID
		var message string
		if isPlayer2Turn {
			message = "É seu turno! Escolha uma carta para jogar."
		} else {
			message = "Carta jogada! Aguardando oponente..."
		}
		
		response, err := protocol.CreateTurnUpdate(currentMatch.ID, message, isPlayer2Turn)
		if err == nil {
			response = append(response, '\n')
			conn2.Write(response)
		}
	}
}

func handleQueue(conn net.Conn, message *protocol.Message) {
	// Extrai os dados da requisição de fila
	queueReq, err := protocol.ExtractQueueRequest(message)
	if err != nil {
		fmt.Println("Erro ao extrair dados de fila:", err)
		return
	}

	fmt.Printf("Tentativa de enfileirar - UserID: %d\n", queueReq.UserID)

	queueMutex.Lock()
	defer queueMutex.Unlock()

	var response []byte
	
	// Verifica se o jogador já está na fila
	playerInQueue := false
	for _, playerID := range queue {
		if playerID == queueReq.UserID {
			playerInQueue = true
			break
		}
	}

	// Verifica se o jogador já está em uma partida
	currentMatch := match.GetManager().GetPlayerMatch(queueReq.UserID)
	if currentMatch != nil {
		response, err = protocol.CreateQueueResponse(false, "Você já está em uma partida!", len(queue))
		fmt.Printf("Jogador %d já está em partida (ID: %d)\n", queueReq.UserID, currentMatch.ID)
	} else if playerInQueue {
		response, err = protocol.CreateQueueResponse(false, "Você já está na fila!", len(queue))
		fmt.Printf("Jogador %d já está na fila\n", queueReq.UserID)
	} else {
		// VALIDAÇÃO ATUALIZADA: Busca o player mais recente e verifica cartas
		var foundPlayer *player.Player
		found := false
		
		// Busca o player atualizado no slice global
		for i := range players {
			if players[i].GetID() == queueReq.UserID {
				foundPlayer = &players[i]
				found = true
				break
			}
		}
		
		if !found {
			response, err = protocol.CreateQueueResponse(false, "Jogador não encontrado!", len(queue))
			fmt.Printf("Jogador %d não encontrado\n", queueReq.UserID)
		} else {
			// Verifica cartas em tempo real
			currentInventorySize := foundPlayer.GetInventorySize()
			hydra, quimera, gorgona := foundPlayer.CountCardsByType()
			
			if currentInventorySize == 0 {
				response, err = protocol.CreateQueueResponse(false, "Você não tem cartas! Abra um pacote primeiro para jogar.", len(queue))
				fmt.Printf("Jogador %d tentou entrar na fila SEM cartas (H:%d Q:%d G:%d)\n", 
					queueReq.UserID, hydra, quimera, gorgona)
			} else {
				// Adiciona o jogador à fila
				queue = append(queue, queueReq.UserID)
				response, err = protocol.CreateQueueResponse(true, "Você foi adicionado à fila de partidas!", len(queue))
				fmt.Printf("Jogador %d adicionado à fila. Total na fila: %d (cartas: H:%d Q:%d G:%d = %d total)\n", 
					queueReq.UserID, len(queue), hydra, quimera, gorgona, currentInventorySize)
			}
		}
	}

	if err != nil {
		fmt.Println("Erro ao criar resposta:", err)
		return
	}

	// Envia a resposta
	response = append(response, '\n')
	_, err = conn.Write(response)
	if err != nil {
		fmt.Println("Erro ao enviar resposta:", err)
	}
}

func handleRegister(conn net.Conn, message *protocol.Message) {
	// Extrai os dados do register request
	registerReq, err := protocol.ExtractRegisterRequest(message)
	if err != nil {
		fmt.Println("Erro ao extrair dados de cadastro:", err)
		return
	}

	fmt.Printf("Tentativa de cadastro - Usuário: %s\n", registerReq.UserName)

	// Validações
	var response []byte
	
	// Verifica se username já existe
	if userExists(registerReq.UserName) {
		response, err = protocol.CreateRegisterResponse(false, "Nome de usuário já existe!", 0)
		fmt.Printf("Cadastro falhou - usuário já existe: %s\n", registerReq.UserName)
	} else if len(strings.TrimSpace(registerReq.UserName)) < 3 {
		response, err = protocol.CreateRegisterResponse(false, "Nome de usuário deve ter pelo menos 3 caracteres!", 0)
		fmt.Printf("Cadastro falhou - username muito curto: %s\n", registerReq.UserName)
	} else if len(strings.TrimSpace(registerReq.Password)) < 4 {
		response, err = protocol.CreateRegisterResponse(false, "Senha deve ter pelo menos 4 caracteres!", 0)
		fmt.Printf("Cadastro falhou - senha muito curta para usuário: %s\n", registerReq.UserName)
	} else {
		// Cria novo player
		newPlayer := player.NewPlayer(nextID, registerReq.UserName, registerReq.Password)
		players = append(players, newPlayer)
		
		response, err = protocol.CreateRegisterResponse(true, "Cadastro realizado com sucesso!", nextID)
		fmt.Printf("Cadastro bem-sucedido - Usuário: %s (ID: %d)\n", registerReq.UserName, nextID)
		
		nextID++
	}

	if err != nil {
		fmt.Println("Erro ao criar resposta:", err)
		return
	}

	// Envia a resposta
	response = append(response, '\n') // Adiciona quebra de linha para o scanner
	_, err = conn.Write(response)
	if err != nil {
		fmt.Println("Erro ao enviar resposta:", err)
	}
}

func handleLogin(conn net.Conn, message *protocol.Message) int {
	// Extrai os dados do login request
	loginReq, err := protocol.ExtractLoginRequest(message)
	if err != nil {
		fmt.Println("Erro ao extrair dados de login:", err)
		return 0
	}

	fmt.Printf("Tentativa de login - Usuário: %s\n", loginReq.UserName)

	// Busca o player na lista
	player, found := findPlayer(loginReq.UserName, loginReq.Password)
	
	var response []byte
	var userID int
	
	if found {
		// Verifica se o usuário já está conectado
		connectedMutex.Lock()
		alreadyConnected := connectedUsers[player.GetID()]
		if !alreadyConnected {
			// Marca como conectado
			connectedUsers[player.GetID()] = true
			userID = player.GetID()
			
			// Armazena a conexão
			connectionsMutex.Lock()
			userConnections[player.GetID()] = conn
			connectionsMutex.Unlock()
		}
		connectedMutex.Unlock()
		
		if alreadyConnected {
			// Usuário já está conectado
			response, err = protocol.CreateLoginResponse(false, "Usuário já está conectado em outra sessão!", 0)
			fmt.Printf("Login negado - usuário %s já está conectado (ID: %d)\n", loginReq.UserName, player.GetID())
		} else {
			// Login bem-sucedido
			response, err = protocol.CreateLoginResponse(true, "Login realizado com sucesso!", player.GetID())
			fmt.Printf("Login bem-sucedido para usuário: %s (ID: %d)\n", loginReq.UserName, player.GetID())
		}
	} else {
		// Login falhou
		response, err = protocol.CreateLoginResponse(false, "Usuário ou senha incorretos!", 0)
		fmt.Printf("Login falhou para usuário: %s\n", loginReq.UserName)
	}

	if err != nil {
		fmt.Println("Erro ao criar resposta:", err)
		return 0
	}

	// Envia a resposta
	response = append(response, '\n') // Adiciona quebra de linha para o scanner
	_, err = conn.Write(response)
	if err != nil {
		fmt.Println("Erro ao enviar resposta:", err)
		return 0
	}
	
	return userID
}

// Função de ping
func handlePing(conn net.Conn, message *protocol.Message) {
	// Extrai os dados da requisição de ping
	pingReq, err := protocol.ExtractPingRequest(message)
	if err != nil {
		fmt.Println("Erro ao extrair dados de ping:", err)
		return
	}

	fmt.Printf("Ping recebido do usuário ID: %d\n", pingReq.UserID)

	var response []byte
	
	// Verifica se o usuário está conectado
	connectedMutex.Lock()
	isConnected := connectedUsers[pingReq.UserID]
	connectedMutex.Unlock()
	
	if !isConnected {
		// Usuário não está conectado/autenticado
		response, err = protocol.CreatePingResponse(false, "Usuário não está conectado!")
		fmt.Printf("Ping negado - usuário %d não está conectado\n", pingReq.UserID)
	} else {
		// Usuário conectado, responde com sucesso
		response, err = protocol.CreatePingResponse(true, "Pong! Servidor respondeu.")
		fmt.Printf("Ping respondido para usuário %d\n", pingReq.UserID)
	}

	if err != nil {
		fmt.Println("Erro ao criar resposta de ping:", err)
		return
	}

	// Envia a resposta
	response = append(response, '\n')
	_, err = conn.Write(response)
	if err != nil {
		fmt.Println("Erro ao enviar resposta de ping:", err)
	}
}

func handleStats(conn net.Conn, message *protocol.Message) {
	// Extrai os dados da requisição de estatísticas
	statsReq, err := protocol.ExtractStatsRequest(message)
	if err != nil {
		fmt.Println("Erro ao extrair dados de estatísticas:", err)
		return
	}

	fmt.Printf("Requisição de estatísticas - UserID: %d\n", statsReq.UserID)

	// Verifica se o usuário está conectado
	connectedMutex.Lock()
	isConnected := connectedUsers[statsReq.UserID]
	connectedMutex.Unlock()

	var response []byte

	if !isConnected {
		// Usuário não está conectado/autenticado
		response, err = protocol.CreateStatsResponse(false, "Usuário não está conectado!", "", 0, 0, 0.0)
		fmt.Printf("Estatísticas negadas - usuário %d não está conectado\n", statsReq.UserID)
	} else {
		// Busca o player
		player, found := findPlayerByID(statsReq.UserID)
		if !found {
			response, err = protocol.CreateStatsResponse(false, "Usuário não encontrado!", "", 0, 0, 0.0)
			fmt.Printf("Estatísticas negadas - usuário %d não encontrado\n", statsReq.UserID)
		} else {
			// Usuário conectado, retorna estatísticas
			wins := player.GetWins()
			losses := player.GetLosses()
			winRate := player.GetWinRate()
			message := "Estatísticas obtidas com sucesso!"
			
			response, err = protocol.CreateStatsResponse(true, message, player.GetUserName(), wins, losses, winRate)
			fmt.Printf("Estatísticas enviadas para usuário %d: %dW-%dL (%.1f%%)\n", 
				statsReq.UserID, wins, losses, winRate)
		}
	}

	if err != nil {
		fmt.Println("Erro ao criar resposta de estatísticas:", err)
		return
	}

	// Envia a resposta
	response = append(response, '\n')
	_, err = conn.Write(response)
	if err != nil {
		fmt.Println("Erro ao enviar resposta de estatísticas:", err)
	}
}

// Função para verificar se usuário já existe
func userExists(userName string) bool {
	for _, player := range players {
		if player.GetUserName() == userName {
			return true
		}
	}
	return false
}

// Função para buscar um player pelos credentials
func findPlayer(userName, password string) (player.Player, bool) {
	for _, player := range players {
		if player.GetUserName() == userName && player.GetPassword() == password {
			return player, true
		}
	}
	return player.Player{}, false
}

// Função para buscar um player pelo ID
func findPlayerByID(playerID int) (*player.Player, bool) {
    for i := range players {
        if players[i].GetID() == playerID {
            return &players[i], true  // Retorna PONTEIRO para o original
        }
    }
    return nil, false
}

func cleanupOrphanedMatches() {
	ticker := time.NewTicker(5 * time.Second) // Verifica a cada 5 segundos
	defer ticker.Stop()

	for range ticker.C {
		// Obtém todas as partidas ativas
		activeMatches := match.GetManager().GetAllActiveMatches()
		
		connectionsMutex.Lock()
		connectedMutex.Lock()
		
		for _, currentMatch := range activeMatches {
			player1ID := currentMatch.Player1.GetID()
			player2ID := currentMatch.Player2.GetID()
			
			player1Connected := connectedUsers[player1ID]
			player2Connected := connectedUsers[player2ID]
			
			// Se ambos jogadores desconectaram, cancela a partida
			if !player1Connected && !player2Connected {
				fmt.Printf("🧹 Partida %d cancelada - ambos jogadores desconectaram\n", currentMatch.ID)
				match.GetManager().CancelMatch(currentMatch.ID)
				continue
			}
			
			// Se apenas um jogador desconectou, declara o outro vencedor
			if !player1Connected && player2Connected {
				fmt.Printf("🏆 Player 2 vence partida %d por desconexão do oponente\n", currentMatch.ID)
				match.GetManager().ForceWin(currentMatch.ID, currentMatch.Player2.GetID())
				
				// Atualiza estatísticas
				updatePlayerStats(currentMatch.Player1.GetID(), currentMatch.Player2.GetID(), currentMatch.Player2.GetID())
				
				// Notifica o jogador restante
				if conn, exists := userConnections[currentMatch.Player2.GetID()]; exists {
					message := "Seu oponente desconectou. Você venceu por abandono!"
					response, _ := protocol.CreateMatchEnd(currentMatch.ID, currentMatch.Player2.GetID(), 
						currentMatch.Player2.GetUserName(), message)
					response = append(response, '\n')
					conn.Write(response)
				}
				
			} else if player1Connected && !player2Connected {
				fmt.Printf("🏆 Player 1 vence partida %d por desconexão do oponente\n", currentMatch.ID)
				match.GetManager().ForceWin(currentMatch.ID, currentMatch.Player1.GetID())
				
				// Atualiza estatísticas
				updatePlayerStats(currentMatch.Player1.GetID(), currentMatch.Player2.GetID(), currentMatch.Player1.GetID())
				
				// Notifica o jogador restante
				if conn, exists := userConnections[currentMatch.Player1.GetID()]; exists {
					message := "Seu oponente desconectou. Você venceu por abandono!"
					response, _ := protocol.CreateMatchEnd(currentMatch.ID, currentMatch.Player1.GetID(), 
						currentMatch.Player1.GetUserName(), message)
					response = append(response, '\n')
					conn.Write(response)
				}
			}
		}
		
		connectedMutex.Unlock()
		connectionsMutex.Unlock()
	}
}