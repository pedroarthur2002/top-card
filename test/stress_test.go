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

// Teste de stress para abertura de pacotes
func TestStressCardPacks(t *testing.T) {
	numUsers := 10
	var wg sync.WaitGroup
	
	pacoteSucesso := 0
	pacoteFalha := 0
	erroConexao := 0
	erroRegistro := 0
	erroLogin := 0
	var mutex sync.Mutex

	startTime := time.Now()

	for i := 0; i < numUsers; i++ {
		wg.Add(1)
		go func(userNum int) {
			defer wg.Done()
			
			username := fmt.Sprintf("pack_user_%d_%d", userNum, time.Now().UnixNano())
			
			conn, err := net.DialTimeout("tcp", "localhost:8080", 5*time.Second)
			if err != nil {
				mutex.Lock()
				erroConexao++
				mutex.Unlock()
				t.Logf("User%d: ERRO CONEXAO - %v", userNum, err)
				return
			}
			defer conn.Close()
			
			conn.SetDeadline(time.Now().Add(15 * time.Second))
			scanner := bufio.NewScanner(conn)
			
			// 1. REGISTRO
			registerMsg, _ := protocol.CreateRegisterRequest(username, "pass123")
			registerMsg = append(registerMsg, '\n')
			if _, err = conn.Write(registerMsg); err != nil {
				mutex.Lock()
				erroConexao++
				mutex.Unlock()
				return
			}
			
			if !scanner.Scan() {
				mutex.Lock()
				erroRegistro++
				mutex.Unlock()
				t.Logf("User%d: ERRO SCAN REGISTRO", userNum)
				return
			}
			
			response, err := protocol.DecodeMessage(scanner.Bytes())
			if err != nil {
				mutex.Lock()
				erroRegistro++
				mutex.Unlock()
				return
			}
			
			registerResp, err := protocol.ExtractRegisterResponse(response)
			if err != nil || !registerResp.Success {
				mutex.Lock()
				erroRegistro++
				mutex.Unlock()
				t.Logf("User%d: ERRO REGISTRO - %s", userNum, registerResp.Message)
				return
			}
			
			userID := registerResp.UserID
			
			// 2. LOGIN
			loginMsg, _ := protocol.CreateLoginRequest(username, "pass123")
			loginMsg = append(loginMsg, '\n')
			if _, err = conn.Write(loginMsg); err != nil {
				mutex.Lock()
				erroLogin++
				mutex.Unlock()
				return
			}
			
			if !scanner.Scan() {
				mutex.Lock()
				erroLogin++
				mutex.Unlock()
				return
			}
			
			response, err = protocol.DecodeMessage(scanner.Bytes())
			if err != nil {
				mutex.Lock()
				erroLogin++
				mutex.Unlock()
				return
			}
			
			loginResp, err := protocol.ExtractLoginResponse(response)
			if err != nil || !loginResp.Success {
				mutex.Lock()
				erroLogin++
				mutex.Unlock()
				t.Logf("User%d: ERRO LOGIN - %s", userNum, loginResp.Message)
				return
			}
			
			// 3. ABRIR PACOTE
			packMsg, _ := protocol.CreateCardPackRequest(userID)
			packMsg = append(packMsg, '\n')
			if _, err = conn.Write(packMsg); err != nil {
				mutex.Lock()
				pacoteFalha++
				mutex.Unlock()
				return
			}
			
			if !scanner.Scan() {
				mutex.Lock()
				pacoteFalha++
				mutex.Unlock()
				return
			}
			
			response, err = protocol.DecodeMessage(scanner.Bytes())
			if err != nil {
				mutex.Lock()
				pacoteFalha++
				mutex.Unlock()
				return
			}
			
			packResp, err := protocol.ExtractCardPackResponse(response)
			if err != nil {
				mutex.Lock()
				pacoteFalha++
				mutex.Unlock()
				return
			}
			
			mutex.Lock()
			if packResp.Success {
				pacoteSucesso++
				t.Logf("User%d: PACOTE ABERTO - %d cartas", userNum, len(packResp.Cards))
			} else {
				pacoteFalha++
				t.Logf("User%d: PACOTE NEGADO - %s", userNum, packResp.Message)
			}
			mutex.Unlock()
			
		}(i+1)
		
		time.Sleep(50 * time.Millisecond)
	}
	
	wg.Wait()
	elapsed := time.Since(startTime)
	
	mutex.Lock()
	t.Logf("\n========================================")
	t.Logf("      TESTE STRESS - PACOTES")
	t.Logf("========================================")
	t.Logf("Usuarios testados: %d", numUsers)
	t.Logf("Tempo total: %v", elapsed)
	t.Logf("----------------------------------------")
	t.Logf("PACOTES ABERTOS:     %d", pacoteSucesso)
	t.Logf("PACOTES NEGADOS:     %d", pacoteFalha)
	t.Logf("ERRO CONEXAO:        %d", erroConexao)
	t.Logf("ERRO REGISTRO:       %d", erroRegistro)
	t.Logf("ERRO LOGIN:          %d", erroLogin)
	t.Logf("----------------------------------------")
	total := pacoteSucesso + pacoteFalha + erroConexao + erroRegistro + erroLogin
	t.Logf("TOTAL:               %d", total)
	
	if pacoteSucesso > 0 {
		successRate := float64(pacoteSucesso) / float64(numUsers) * 100
		t.Logf("TAXA SUCESSO:        %.1f%%", successRate)
	}
	
	t.Logf("========================================")
	mutex.Unlock()
}

// Teste de stress para matchmaking
func TestStressMatchmaking(t *testing.T) {
	numUsers := 2 // Número par para formar partidas
	var wg sync.WaitGroup
	
	sucessoCompleto := 0
	erroConexao := 0
	erroRegistro := 0
	erroLogin := 0
	erroPacote := 0
	erroFila := 0
	partidasEncontradas := 0
	timeoutPartida := 0
	var mutex sync.Mutex

	startTime := time.Now()

	for i := 0; i < numUsers; i++ {
		wg.Add(1)
		go func(userNum int) {
			defer wg.Done()
			
			username := fmt.Sprintf("match_player_%d_%d", userNum, time.Now().UnixNano())
			
			conn, err := net.DialTimeout("tcp", "localhost:8080", 5*time.Second)
			if err != nil {
				mutex.Lock()
				erroConexao++
				mutex.Unlock()
				t.Logf("Player%d: ERRO CONEXAO", userNum)
				return
			}
			defer conn.Close()
			
			conn.SetDeadline(time.Now().Add(60 * time.Second)) // Timeout aumentado para 60s
			scanner := bufio.NewScanner(conn)
			
			// 1. REGISTRO
			registerMsg, _ := protocol.CreateRegisterRequest(username, "pass123")
			registerMsg = append(registerMsg, '\n')
			if _, err = conn.Write(registerMsg); err != nil {
				mutex.Lock()
				erroRegistro++
				mutex.Unlock()
				return
			}
			
			if !scanner.Scan() {
				mutex.Lock()
				erroRegistro++
				mutex.Unlock()
				return
			}
			
			response, err := protocol.DecodeMessage(scanner.Bytes())
			if err != nil {
				mutex.Lock()
				erroRegistro++
				mutex.Unlock()
				return
			}
			
			registerResp, err := protocol.ExtractRegisterResponse(response)
			if err != nil || !registerResp.Success {
				mutex.Lock()
				erroRegistro++
				mutex.Unlock()
				t.Logf("Player%d: ERRO REGISTRO", userNum)
				return
			}
			userID := registerResp.UserID
			
			// 2. LOGIN
			loginMsg, _ := protocol.CreateLoginRequest(username, "pass123")
			loginMsg = append(loginMsg, '\n')
			if _, err = conn.Write(loginMsg); err != nil {
				mutex.Lock()
				erroLogin++
				mutex.Unlock()
				return
			}
			
			if !scanner.Scan() {
				mutex.Lock()
				erroLogin++
				mutex.Unlock()
				return
			}
			
			response, err = protocol.DecodeMessage(scanner.Bytes())
			if err != nil {
				mutex.Lock()
				erroLogin++
				mutex.Unlock()
				return
			}
			
			loginResp, err := protocol.ExtractLoginResponse(response)
			if err != nil || !loginResp.Success {
				mutex.Lock()
				erroLogin++
				mutex.Unlock()
				t.Logf("Player%d: ERRO LOGIN", userNum)
				return
			}
			
			// 3. ABRIR PACOTE (necessário para entrar na fila)
			packMsg, _ := protocol.CreateCardPackRequest(userID)
			packMsg = append(packMsg, '\n')
			if _, err = conn.Write(packMsg); err != nil {
				mutex.Lock()
				erroPacote++
				mutex.Unlock()
				return
			}
			
			if !scanner.Scan() {
				mutex.Lock()
				erroPacote++
				mutex.Unlock()
				return
			}
			
			response, err = protocol.DecodeMessage(scanner.Bytes())
			if err != nil {
				mutex.Lock()
				erroPacote++
				mutex.Unlock()
				return
			}
			
			packResp, err := protocol.ExtractCardPackResponse(response)
			if err != nil || !packResp.Success {
				mutex.Lock()
				erroPacote++
				mutex.Unlock()
				t.Logf("Player%d: ERRO PACOTE", userNum)
				return
			}
			
			// 4. ENTRAR NA FILA
			queueMsg, _ := protocol.CreateQueueRequest(userID)
			queueMsg = append(queueMsg, '\n')
			if _, err = conn.Write(queueMsg); err != nil {
				mutex.Lock()
				erroFila++
				mutex.Unlock()
				return
			}
			
			if !scanner.Scan() {
				mutex.Lock()
				erroFila++
				mutex.Unlock()
				return
			}
			
			response, err = protocol.DecodeMessage(scanner.Bytes())
			if err != nil {
				mutex.Lock()
				erroFila++
				mutex.Unlock()
				return
			}
			
			queueResp, err := protocol.ExtractQueueResponse(response)
			if err != nil || !queueResp.Success {
				mutex.Lock()
				erroFila++
				mutex.Unlock()
				t.Logf("Player%d: ERRO FILA - %s", userNum, queueResp.Message)
				return
			}
			
			t.Logf("Player%d: NA FILA - tamanho: %d", userNum, queueResp.QueueSize)
			
			// 5. AGUARDAR PARTIDA (aguarda até 30 segundos)
			timeout := time.NewTimer(30 * time.Second) // Aumentado de 25s para 30s
			defer timeout.Stop()
			matchFound := false
			messagesReceived := 0
			
			// Remove timeout específico - usa timeout geral da conexão
			conn.SetReadDeadline(time.Time{}) // Remove deadline específico
			
			for !matchFound && messagesReceived < 10 { // Aumentado limite de mensagens
				select {
				case <-timeout.C:
					mutex.Lock()
					timeoutPartida++
					mutex.Unlock()
					t.Logf("Player%d: TIMEOUT - partida não encontrada após 30s", userNum)
					return
				default:
					// Canal para fazer leitura não-bloqueante
					scanChan := make(chan bool, 1)
					go func() {
						scanChan <- scanner.Scan()
					}()
					
					select {
					case scanned := <-scanChan:
						if scanned {
							messagesReceived++
							t.Logf("Player%d: Mensagem recebida (%d/10)", userNum, messagesReceived)
							
							response, err := protocol.DecodeMessage(scanner.Bytes())
							if err == nil {
								t.Logf("Player%d: Tipo mensagem: %s", userNum, response.Type)
								switch response.Type {
								case protocol.MSG_MATCH_FOUND:
									mutex.Lock()
									partidasEncontradas++
									mutex.Unlock()
									matchFound = true
									t.Logf("Player%d: PARTIDA ENCONTRADA!", userNum)
								case protocol.MSG_MATCH_START:
									t.Logf("Player%d: PARTIDA INICIADA!", userNum)
								case protocol.MSG_GAME_STATE:
									t.Logf("Player%d: ESTADO DO JOGO recebido!", userNum)
								default:
									t.Logf("Player%d: Mensagem não relacionada: %s", userNum, response.Type)
								}
							} else {
								t.Logf("Player%d: Erro ao decodificar: %v", userNum, err)
							}
						} else {
							if scanner.Err() != nil {
								t.Logf("Player%d: Erro no scanner: %v", userNum, scanner.Err())
								return
							}
							// Scanner retornou false - possivelmente conexão fechada
							time.Sleep(100 * time.Millisecond)
						}
					case <-time.After(1 * time.Second): // Timeout para esta leitura específica
						// Continua tentando se não conseguiu ler
						time.Sleep(100 * time.Millisecond)
					}
				}
			}
			
			if matchFound {
				mutex.Lock()
				sucessoCompleto++
				mutex.Unlock()
			}
			
		}(i+1)
		
		time.Sleep(200 * time.Millisecond)
	}
	
	wg.Wait()
	elapsed := time.Since(startTime)
	
	mutex.Lock()
	t.Logf("\n========================================")
	t.Logf("      TESTE STRESS - MATCHMAKING")
	t.Logf("========================================")
	t.Logf("Players testados: %d", numUsers)
	t.Logf("Tempo total: %v", elapsed)
	t.Logf("----------------------------------------")
	t.Logf("FLUXO COMPLETO:      %d", sucessoCompleto)
	t.Logf("PARTIDAS FORMADAS:   %d", partidasEncontradas/2)
	t.Logf("TIMEOUTS PARTIDA:    %d", timeoutPartida)
	t.Logf("ERRO CONEXAO:        %d", erroConexao)
	t.Logf("ERRO REGISTRO:       %d", erroRegistro)
	t.Logf("ERRO LOGIN:          %d", erroLogin)
	t.Logf("ERRO PACOTE:         %d", erroPacote)
	t.Logf("ERRO FILA:           %d", erroFila)
	t.Logf("----------------------------------------")
	
	if sucessoCompleto > 0 {
		successRate := float64(sucessoCompleto) / float64(numUsers) * 100
		t.Logf("TAXA SUCESSO:        %.1f%%", successRate)
	}
	
	t.Logf("========================================")
	mutex.Unlock()
}