package server

import (
	"bufio"
	"fmt"
	"net"
	"strings"
	"sync"
	"top-card/player"
	"top-card/protocol"
)

var players []Player.Player
var nextID = 1
var queue []int // Fila de jogadores esperando partida
var queueMutex sync.Mutex
var connectedUsers = make(map[int]bool) // Mapa para rastrear usuários conectados por ID
var connectedMutex sync.Mutex           // Mutex para proteger acesso concurrent ao mapa

func Run() {
	// CriaÃ§Ã£o do servidor (ouvindo na porta 8080)
	ln, err := net.Listen("tcp", ":8080")
	if err != nil {
		fmt.Println("Erro do tipo: ", err)
		return
	}
	defer ln.Close()

	fmt.Println("Servidor TOP CARD ouvindo na porta 8080...")

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

func handleConnection(conn net.Conn) {
	defer conn.Close()
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
		default:
			fmt.Println("Tipo de mensagem nÃ£o reconhecido:", message.Type)
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

	if playerInQueue {
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

	fmt.Printf("Tentativa de cadastro - UsuÃ¡rio: %s\n", registerReq.UserName)

	// ValidaÃ§Ãµes
	var response []byte
	
	// Verifica se username jÃ¡ existe
	if userExists(registerReq.UserName) {
		response, err = protocol.CreateRegisterResponse(false, "Nome de usuÃ¡rio jÃ¡ existe!", 0)
		fmt.Printf("Cadastro falhou - usuÃ¡rio jÃ¡ existe: %s\n", registerReq.UserName)
	} else if len(strings.TrimSpace(registerReq.UserName)) < 3 {
		response, err = protocol.CreateRegisterResponse(false, "Nome de usuÃ¡rio deve ter pelo menos 3 caracteres!", 0)
		fmt.Printf("Cadastro falhou - username muito curto: %s\n", registerReq.UserName)
	} else if len(strings.TrimSpace(registerReq.Password)) < 4 {
		response, err = protocol.CreateRegisterResponse(false, "Senha deve ter pelo menos 4 caracteres!", 0)
		fmt.Printf("Cadastro falhou - senha muito curta para usuÃ¡rio: %s\n", registerReq.UserName)
	} else {
		// Cria novo player
		newPlayer := Player.NewPlayer(nextID, registerReq.UserName, registerReq.Password)
		players = append(players, newPlayer)
		
		response, err = protocol.CreateRegisterResponse(true, "Cadastro realizado com sucesso!", nextID)
		fmt.Printf("Cadastro bem-sucedido - UsuÃ¡rio: %s (ID: %d)\n", registerReq.UserName, nextID)
		
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

	fmt.Printf("Tentativa de login - UsuÃ¡rio: %s\n", loginReq.UserName)

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
		}
		connectedMutex.Unlock()
		
		if alreadyConnected {
			// Usuário já está conectado
			response, err = protocol.CreateLoginResponse(false, "Usuário já está conectado em outra sessão!", 0)
			fmt.Printf("Login negado - usuário %s já está conectado (ID: %d)\n", loginReq.UserName, player.GetID())
		} else {
			// Login bem-sucedido
			response, err = protocol.CreateLoginResponse(true, "Login realizado com sucesso!", player.GetID())
			fmt.Printf("Login bem-sucedido para usuÃ¡rio: %s (ID: %d)\n", loginReq.UserName, player.GetID())
		}
	} else {
		// Login falhou
		response, err = protocol.CreateLoginResponse(false, "UsuÃ¡rio ou senha incorretos!", 0)
		fmt.Printf("Login falhou para usuÃ¡rio: %s\n", loginReq.UserName)
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

// FunÃ§Ã£o para verificar se usuÃ¡rio jÃ¡ existe
func userExists(userName string) bool {
	for _, player := range players {
		if player.GetUserName() == userName {
			return true
		}
	}
	return false
}

// FunÃ§Ã£o para buscar um player pelos credentials
func findPlayer(userName, password string) (Player.Player, bool) {
	for _, player := range players {
		if player.GetUserName() == userName && player.GetPassword() == password {
			return player, true
		}
	}
	return Player.Player{}, false
}