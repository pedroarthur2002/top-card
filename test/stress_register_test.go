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

func TestStressRegister(t *testing.T) {
	const (
		SERVER_ADDR = "localhost:8080"
		CLIENTS     = 1000
		REQS_EACH   = 1
	)

	stats := &TestStats{}
	var wg sync.WaitGroup
	start := time.Now()

	for i := 0; i < CLIENTS; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			runClient(t, SERVER_ADDR, id, REQS_EACH, stats)
		}(i)
	}

	wg.Wait()
	duration := time.Since(start)
	total := stats.Success + stats.Failed + stats.Errors

	t.Logf("Resultados (%.2fs):", duration.Seconds())
	t.Logf("Sucessos: %d", stats.Success)
	t.Logf("Falhas: %d", stats.Failed)
	t.Logf("Erros: %d", stats.Errors)
	t.Logf("Taxa sucesso: %.1f%%", float64(stats.Success)/float64(total)*100)
	t.Logf("Req/seg: %.1f", float64(total)/duration.Seconds())

	if stats.Success == 0 {
		t.Fatal("Nenhum cadastro bem-sucedido")
	}
}

func runClient(t *testing.T, addr string, clientID, numReqs int, stats *TestStats) {
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		for i := 0; i < numReqs; i++ {
			stats.AddError()
		}
		return
	}
	defer conn.Close()

	scanner := bufio.NewScanner(conn)

	for i := 0; i < numReqs; i++ {
		username := fmt.Sprintf("user_%d_%d", clientID, i)
		
		if !sendRegister(conn, username, "pass1234") {
			stats.AddError()
			continue
		}

		if !readResponse(scanner, stats) {
			stats.AddError()
		}
	}
}

func sendRegister(conn net.Conn, username, password string) bool {
	// Usa a função do protocolo para criar a mensagem de registro
	registerMessage, err := protocol.CreateRegisterRequest(username, password)
	if err != nil {
		return false
	}

	// Adiciona quebra de linha conforme esperado pelo servidor
	registerMessage = append(registerMessage, '\n')

	// Envia a mensagem
	_, err = conn.Write(registerMessage)
	return err == nil
}

func readResponse(scanner *bufio.Scanner, stats *TestStats) bool {
	done := make(chan bool, 1)
	
	go func() {
		done <- scanner.Scan()
	}()

	select {
	case success := <-done:
		if !success {
			return false
		}

		// Usa a função do protocolo para decodificar a mensagem
		responseData := scanner.Bytes()
		message, err := protocol.DecodeMessage(responseData)
		if err != nil {
			stats.AddError()
			return false
		}

		// Verifica se é resposta de registro
		if message.Type == protocol.MSG_REGISTER_RESPONSE {
			// Usa a função do protocolo para extrair os dados da resposta
			registerResp, err := protocol.ExtractRegisterResponse(message)
			if err != nil {
				stats.AddError()
				return false
			}

			if registerResp.Success {
				stats.AddSuccess()
			} else {
				stats.AddFailed()
			}
		} else {
			stats.AddError()
		}
		return true

	case <-time.After(3 * time.Second):
		return false
	}
}