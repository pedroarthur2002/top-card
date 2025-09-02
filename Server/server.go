package server

import (
	"bufio"
	"fmt"
	"net"
	"strings"
	"top-card/player"
	"top-card/protocol"
)

var players []Player.Player
var nextID = 1

func Run() {
	// Criação do servidor (ouvindo na porta 8080)
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
		default:
			fmt.Println("Tipo de mensagem não reconhecido:", message.Type)
		}
	}

	if err := scanner.Err(); err != nil {
		fmt.Println("Erro ao ler do cliente:", err)
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

func handleLogin(conn net.Conn, message *protocol.Message) {
	// Extrai os dados do login request
	loginReq, err := protocol.ExtractLoginRequest(message)
	if err != nil {
		fmt.Println("Erro ao extrair dados de login:", err)
		return
	}

	fmt.Printf("Tentativa de login - Usuário: %s\n", loginReq.UserName)

	// Busca o player na lista
	player, found := findPlayer(loginReq.UserName, loginReq.Password)
	
	var response []byte
	if found {
		// Login bem-sucedido
		response, err = protocol.CreateLoginResponse(true, "Login realizado com sucesso!", player.GetID())
		fmt.Printf("Login bem-sucedido para usuário: %s (ID: %d)\n", loginReq.UserName, player.GetID())
	} else {
		// Login falhou
		response, err = protocol.CreateLoginResponse(false, "Usuário ou senha incorretos!", 0)
		fmt.Printf("Login falhou para usuário: %s\n", loginReq.UserName)
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