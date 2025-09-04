package server

import (
	"bufio"
	"fmt"
	"net"
	"strings"
	"sync"
	"time"
	"top-card/Player"
	"top-card/protocol"
	"top-card/match"
)

var players []Player.Player
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
	
	// Notifica jogador 1
	connectionsMutex.Lock()
	conn1, exists1 := userConnections[player1ID]
	connectionsMutex.Unlock()
	
	if exists1 {
		response, err := protocol.CreateMatchStart(matchID, message)
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
		response, err := protocol.CreateMatchStart(matchID, message)
		if err == nil {
			response = append(response, '\n')
			conn2.Write(response)
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
		default:
			fmt.Println("Tipo de mensagem não reconhecido:", message.Type)
		}
	}

	if err := scanner.Err(); err != nil {
		fmt.Println("Erro ao ler do cliente:", err)
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
		// Adiciona o jogador à fila
		queue = append(queue, queueReq.UserID)
		response, err = protocol.CreateQueueResponse(true, "Você foi adicionado à fila de partidas!", len(queue))
		fmt.Printf("Jogador %d adicionado à fila. Total na fila: %d\n", queueReq.UserID, len(queue))
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
		newPlayer := Player.NewPlayer(nextID, registerReq.UserName, registerReq.Password)
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
func findPlayer(userName, password string) (Player.Player, bool) {
	for _, player := range players {
		if player.GetUserName() == userName && player.GetPassword() == password {
			return player, true
		}
	}
	return Player.Player{}, false
}

// Função para buscar um player pelo ID
func findPlayerByID(playerID int) (Player.Player, bool) {
	for _, player := range players {
		if player.GetID() == playerID {
			return player, true
		}
	}
	return Player.Player{}, false
}