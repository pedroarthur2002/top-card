package test

import (
	"bufio"
	"fmt"
	"net"
	"sync"
	"testing"
	"time"
	"top-card/internal/protocol"
)

func TestStressLogin(t *testing.T) {
	const (
		SERVER_ADDR = "localhost:8080"
		CLIENTS     = 100
	)

	// Primeiro, cria usuários para fazer login
	setupUsers := CLIENTS
	t.Logf("Criando %d usuários para teste de login...", setupUsers)
	
	var setupWg sync.WaitGroup
	for i := 0; i < setupUsers; i++ {
		setupWg.Add(1)
		go func(id int) {
			defer setupWg.Done()
			createUser(SERVER_ADDR, id)
		}(i)
	}
	setupWg.Wait()
	
	t.Logf("Usuários criados. Aguardando 1 segundo...")
	time.Sleep(1 * time.Second)

	// Agora faz todos os usuários logarem simultaneamente
	t.Logf("Iniciando login simultâneo de %d usuários...", CLIENTS)
	
	stats := &TestStats{}
	var wg sync.WaitGroup
	start := time.Now()

	for i := 0; i < CLIENTS; i++ {
		wg.Add(1)
		go func(userID int) {
			defer wg.Done()
			loginUser(SERVER_ADDR, userID, stats)
		}(i)
	}

	wg.Wait()
	duration := time.Since(start)
	total := stats.Success + stats.Failed + stats.Errors

	t.Logf("Resultados do Login Simultâneo (%.2fs):", duration.Seconds())
	t.Logf("Sucessos: %d", stats.Success)
	t.Logf("Falhas: %d", stats.Failed)
	t.Logf("Erros: %d", stats.Errors)
	t.Logf("Taxa sucesso: %.1f%%", float64(stats.Success)/float64(total)*100)
	t.Logf("Logins/seg: %.1f", float64(total)/duration.Seconds())

	if stats.Success == 0 {
		t.Fatal("Nenhum login bem-sucedido")
	}
}

func createUser(addr string, userID int) {
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		return
	}
	defer conn.Close()

	username := fmt.Sprintf("user_%d", userID)
	password := "pass123"

	// Cria usuário
	registerMessage, err := protocol.CreateRegisterRequest(username, password)
	if err != nil {
		return
	}

	registerMessage = append(registerMessage, '\n')
	conn.Write(registerMessage)
	
	// Lê resposta (não processa, só consome)
	scanner := bufio.NewScanner(conn)
	scanner.Scan()
}

func loginUser(addr string, userID int, stats *TestStats) {
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		stats.AddError()
		return
	}
	defer conn.Close()

	username := fmt.Sprintf("user_%d", userID)
	password := "pass123"

	// Faz login
	loginMessage, err := protocol.CreateLoginRequest(username, password)
	if err != nil {
		stats.AddError()
		return
	}

	loginMessage = append(loginMessage, '\n')
	_, err = conn.Write(loginMessage)
	if err != nil {
		stats.AddError()
		return
	}

	// Lê resposta com timeout
	scanner := bufio.NewScanner(conn)
	done := make(chan bool, 1)
	
	go func() {
		done <- scanner.Scan()
	}()

	select {
	case success := <-done:
		if !success {
			stats.AddError()
			return
		}

		responseData := scanner.Bytes()
		message, err := protocol.DecodeMessage(responseData)
		if err != nil {
			stats.AddError()
			return
		}

		if message.Type == protocol.MSG_LOGIN_RESPONSE {
			loginResp, err := protocol.ExtractLoginResponse(message)
			if err != nil {
				stats.AddError()
				return
			}

			if loginResp.Success {
				stats.AddSuccess()
			} else {
				stats.AddFailed()
			}
		} else {
			stats.AddError()
		}

	case <-time.After(5 * time.Second):
		stats.AddError()
	}
}