package client

import (
	"bufio"
	"net"
	"fmt"
	"os"
	"strings"
	"strconv"
	"time"
	"top-card/protocol"
)

var currentUserID int
var isLoggedIn bool
var inMatch bool // Flag para indicar se está em partida

// Canais para comunicação entre goroutines
var syncResponseChan = make(chan []byte, 10)
var asyncMessageChan = make(chan []byte, 10)

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

	// Inicia a goroutine para distribuir mensagens do servidor
	go messageDistributor(conn)
	
	// Inicia a goroutine para processar mensagens assíncronas
	go asyncMessageProcessor()

	reader := bufio.NewReader(os.Stdin)
	
	for {
		fmt.Println("\n========================")
		fmt.Println("Bem vindo ao TOP CARD!")
		fmt.Println("========================")
		if isLoggedIn {
			fmt.Printf("Logado como ID: %d\n", currentUserID)
			if inMatch {
				fmt.Println("🎮 Você está atualmente em uma partida!")
			}
		}
		fmt.Println("1 - Fazer login")
		fmt.Println("2 - Cadastrar-se")
		fmt.Println("3 - Abrir pacote de cartas")
		fmt.Println("4 - Buscar partida")
		fmt.Println("5 - Verificar ping")
		fmt.Println("6 - Sair")
		
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
			fmt.Println("Funcionalidade ainda não implementada...") // Fazer a abertura de pacotes

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
			fmt.Println("Você escolheu sair. Saindo...")
			return
			
		default:
			fmt.Println("Opção inválida!")
			// Limpa o buffer em caso de entrada inválida
			reader.ReadString('\n')
		}
	}
}

// Goroutine que distribui mensagens entre síncronas e assíncronas
func messageDistributor(conn net.Conn) {
	serverReader := bufio.NewScanner(conn)
	
	for serverReader.Scan() {
		responseData := serverReader.Bytes()
		
		// Cria uma cópia dos dados para não ter problemas de referência
		dataCopy := make([]byte, len(responseData))
		copy(dataCopy, responseData)
		
		// Decodifica a mensagem para verificar o tipo
		message, err := protocol.DecodeMessage(dataCopy)
		if err != nil {
			fmt.Printf("\n🔴 Erro ao decodificar mensagem do servidor: %v\n", err)
			continue
		}

		// Distribui mensagens baseado no tipo
		switch message.Type {
		case protocol.MSG_LOGIN_RESPONSE, protocol.MSG_REGISTER_RESPONSE, protocol.MSG_QUEUE_RESPONSE, protocol.MSG_PING_RESPONSE:
			// Mensagens síncronas - envia para canal síncrono
			select {
			case syncResponseChan <- dataCopy:
			case <-time.After(100 * time.Millisecond):
				fmt.Printf("\n⚠️  Timeout ao enviar resposta síncrona\n")
			}
		case protocol.MSG_MATCH_FOUND, protocol.MSG_MATCH_START, protocol.MSG_MATCH_END:
			// Mensagens assíncronas - envia para canal assíncrono
			select {
			case asyncMessageChan <- dataCopy:
			case <-time.After(100 * time.Millisecond):
				fmt.Printf("\n⚠️  Timeout ao enviar mensagem assíncrona\n")
			}
		default:
			fmt.Printf("\n⚠️  Tipo de mensagem desconhecido: %s\n", message.Type)
		}
	}

	if err := serverReader.Err(); err != nil {
		fmt.Printf("\n🔴 Erro ao ler mensagens do servidor: %v\n", err)
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
		return nil, fmt.Errorf("timeout aguardando resposta do servidor")
	}
}

// Manipula notificação de partida encontrada
func handleMatchFound(message *protocol.Message) {
	matchFound, err := protocol.ExtractMatchFound(message)
	if err != nil {
		fmt.Printf("\n🔴 Erro ao extrair dados de partida encontrada: %v\n", err)
		return
	}

	fmt.Printf("\n\n🎯 ===== PARTIDA ENCONTRADA! =====\n")
	fmt.Printf("🎮 Match ID: %d\n", matchFound.MatchID)
	fmt.Printf("⚔️  Oponente: %s (ID: %d)\n", matchFound.OpponentName, matchFound.OpponentID)
	fmt.Printf("📝 %s\n", matchFound.Message)
	fmt.Printf("⏳ Preparando a partida...\n")
	fmt.Printf("==================================\n")
	fmt.Print("Pressione Enter para continuar...")
	
	inMatch = true
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
	fmt.Printf("⚔️  Que comece a batalha!\n")
	fmt.Printf("===============================\n")
	fmt.Print("Pressione Enter para continuar...")
	
	inMatch = true
}

// Manipula notificação de fim de partida
func handleMatchEnd(message *protocol.Message) {
	matchEnd, err := protocol.ExtractMatchEnd(message)
	if err != nil {
		fmt.Printf("\n🔴 Erro ao extrair dados de fim de partida: %v\n", err)
		return
	}

	fmt.Printf("\n\n🏁 ===== PARTIDA FINALIZADA! =====\n")
	fmt.Printf("🎮 Match ID: %d\n", matchEnd.MatchID)
	
	if matchEnd.WinnerID == currentUserID {
		fmt.Printf("🎉 VITÓRIA! Você ganhou!\n")
	} else {
		fmt.Printf("😔 DERROTA! Vencedor: %s (ID: %d)\n", matchEnd.WinnerName, matchEnd.WinnerID)
	}
	
	fmt.Printf("📝 %s\n", matchEnd.Message)
	fmt.Printf("🔄 Voltando ao menu principal...\n")
	fmt.Printf("=================================\n")
	fmt.Print("Pressione Enter para continuar...")
	
	inMatch = false
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

// Função de ping
func handlePing(conn net.Conn) {
	if !isLoggedIn {
		fmt.Println("❌ Você precisa estar logado para verificar o ping!")
		return
	}
	
	if currentUserID <= 0 {
		fmt.Println("❌ ID de usuário inválido!")
		return
	}

	fmt.Println("\n--- CONSULTA DE PING ---")
	fmt.Println("🏓 Verificando latência...")

	// Registra o tempo de envio
	startTime := time.Now()

	// Cria a mensagem de requisição do ping
	pingMessage, err := protocol.CreatePingRequest(currentUserID)
	if err != nil {
		fmt.Println("Erro ao solicitar o ping:", err)
		return
	}

	// Adiciona quebra de linha para o servidor conseguir ler
	pingMessage = append(pingMessage, '\n')

	// Envia para o servidor
	_, err = conn.Write(pingMessage)
	if err != nil {
		fmt.Println("Erro ao enviar requisição de ping:", err)
		return
	}

	// Aguarda resposta síncrona
	responseData, err := waitForSyncResponse(5 * time.Second)
	if err != nil {
		fmt.Println("Erro:", err)
		return
	}

	// Registra o tempo de recebimento
	endTime := time.Now()

	// Decodifica a resposta
	message, err := protocol.DecodeMessage(responseData)
	if err != nil {
		fmt.Println("Erro ao decodificar resposta:", err)
		return
	}

	if message.Type == protocol.MSG_PING_RESPONSE {
		pingResp, err := protocol.ExtractPingResponse(message)
		if err != nil {
			fmt.Println("Erro ao extrair resposta do ping:", err)
			return
		}

		if pingResp.Success {
			// Calcula a latência
			latency := endTime.Sub(startTime).Milliseconds()
			
			fmt.Printf("✅ %s\n", pingResp.Message)
			fmt.Printf("🏓 Latência (round-trip): %d ms\n", latency)
		} else {
			fmt.Printf("❌ %s\n", pingResp.Message)
		}
	} else {
		fmt.Printf("⚠️  Tipo de resposta inesperado: %s\n", message.Type)
	}
}