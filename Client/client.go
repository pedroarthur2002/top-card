package client

import (
	"bufio"
	"net"
	"fmt"
	"os"
	"strings"
	"top-card/protocol"
)

var currentUserID int
var isLoggedIn bool

func Run() {

	serverAddr := os.Getenv("SERVER_ADDR")
	if serverAddr == "" {
		serverAddr = "localhost:8080" // Default se não estiver definido
	}

	conn, err := net.Dial("tcp", serverAddr)

	if err != nil {
		fmt.Println("Erro ao conectar no servidor:", err)
		return
	}
	defer conn.Close()

	fmt.Println("Conectado ao servidor TOP CARD!")

	reader := bufio.NewReader(os.Stdin)
	
	for {
		fmt.Println("\n========================")
		fmt.Println("Bem vindo ao TOP CARD!")
		fmt.Println("========================")
		if isLoggedIn {
			fmt.Printf("Logado como ID: %d\n", currentUserID)
		}
		fmt.Println("1 - Fazer login")
		fmt.Println("2 - Cadastrar-se")
		fmt.Println("3 - Abrir pacote de cartas")
		fmt.Println("4 - Buscar partida")
		fmt.Println("5 - Sair")
		
		fmt.Print("Insira sua opção: ")
		var choice int
		fmt.Scanf("%d", &choice)
		reader.ReadString('\n') // Limpa o buffer depois do scanf

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

		case 4:
			if !isLoggedIn {
				fmt.Println("Você precisa estar logado para buscar partida!")
				continue
			}
			handleQueue(conn)
			
		case 5:
			fmt.Println("Você escolheu sair. Saindo...")
			return
			
		default:
			fmt.Println("Opção inválida!")
			// Limpa o buffer em caso de entrada inválida
			reader.ReadString('\n')
		}
	}
}

func handleQueue(conn net.Conn) {
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

	// Lê a resposta do servidor
	serverReader := bufio.NewScanner(conn)
	if serverReader.Scan() {
		responseData := serverReader.Bytes()
		
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
				fmt.Println("Aguardando por oponentes...")
			} else {
				fmt.Printf("❌ %s\n", queueResp.Message)
			}
		}
	} else {
		fmt.Println("Erro ao ler resposta do servidor")
	}
}

func handleRegister(conn net.Conn, reader *bufio.Reader) {
	fmt.Println("\n--- CADASTRO ---")
	fmt.Print("Insira um nome de usuário (máx. 3 caracteres): ")
	userName, _ := reader.ReadString('\n')
	userName = strings.TrimSpace(userName)
	
	fmt.Print("Digite sua senha (máx. 4 caracteres): ")
	password, _ := reader.ReadString('\n')
	password = strings.TrimSpace(password)

	// ValidaÃ§Ãµes bÃ¡sicas no cliente
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

	// LÃª a resposta do servidor
	serverReader := bufio.NewScanner(conn)
	if serverReader.Scan() {
		responseData := serverReader.Bytes()
		
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
				fmt.Printf("Seu ID de usuário Ã©: %d\n", registerResp.UserID)
				fmt.Println("Agora você pode fazer login!")
			} else {
				fmt.Printf("❌ %s\n", registerResp.Message)
			}
		}
	} else {
		fmt.Println("Erro ao ler resposta do servidor")
	}
}

func handleLogin(conn net.Conn, reader *bufio.Reader) {
	fmt.Println("\n--- LOGIN ---")
	fmt.Print("Insira seu nome de usuÃ¡rio: ")
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

	// LÃª a resposta do servidor
	serverReader := bufio.NewScanner(conn)
	if serverReader.Scan() {
		responseData := serverReader.Bytes()
		
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
				fmt.Printf("VocÃª estÃ¡ logado com ID: %d\n", currentUserID)
			} else {
				fmt.Printf("❌ %s\n", loginResp.Message)
			}
		}
	} else {
		fmt.Println("Erro ao ler resposta do servidor")
	}
}