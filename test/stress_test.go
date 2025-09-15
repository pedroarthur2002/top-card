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

// Teste de stress para matchmaking completo (incluindo jogadas)
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
	jogadasRealizadas := 0
	partidasCompletas := 0
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
			
			conn.SetDeadline(time.Now().Add(60 * time.Second)) // Aumentado para 60s para permitir jogadas
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
			
			// 5. AGUARDAR PARTIDA E FAZER JOGADAS
			timeout := time.NewTimer(60 * time.Second)
			defer timeout.Stop()
			matchFound := false
			matchID := 0
			messagesReceived := 0
			myTurn := false
			gameOver := false
			
			// Remove timeout específico - usa timeout geral da conexão
			conn.SetReadDeadline(time.Time{})
			
			for !gameOver && messagesReceived < 20 { // Aumentado limite de mensagens
				select {
				case <-timeout.C:
					mutex.Lock()
					timeoutPartida++
					mutex.Unlock()
					t.Logf("Player%d: TIMEOUT - partida não completada após 60s", userNum)
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
							response, err := protocol.DecodeMessage(scanner.Bytes())
							if err != nil {
								t.Logf("Player%d: Erro ao decodificar mensagem %d: %v", userNum, messagesReceived, err)
								continue
							}
							
							t.Logf("Player%d: Mensagem %d - Tipo: %s", userNum, messagesReceived, response.Type)
							
							switch response.Type {
							case protocol.MSG_MATCH_FOUND:
								matchFoundData, err := protocol.ExtractMatchFound(response)
								if err == nil {
									matchID = matchFoundData.MatchID
									matchFound = true
									mutex.Lock()
									partidasEncontradas++
									mutex.Unlock()
									t.Logf("Player%d: PARTIDA ENCONTRADA! MatchID: %d", userNum, matchID)
								}
							case protocol.MSG_MATCH_START:
								t.Logf("Player%d: PARTIDA INICIADA!", userNum)
							case protocol.MSG_GAME_STATE:
								gameState, err := protocol.ExtractGameState(response)
								if err == nil {
									myTurn = gameState.YourTurn
									gameOver = gameState.GameOver
									if gameOver {
										t.Logf("Player%d: JOGO FINALIZADO", userNum)
										mutex.Lock()
										partidasCompletas++
										mutex.Unlock()
									}
								}
							case protocol.MSG_TURN_UPDATE:
								turnUpdate, err := protocol.ExtractTurnUpdate(response)
								if err == nil {
									myTurn = turnUpdate.YourTurn
									t.Logf("Player%d: TURNO ATUALIZADO - Meu turno: %v", userNum, myTurn)
								}
							case protocol.MSG_MATCH_END:
								gameOver = true
								mutex.Lock()
								partidasCompletas++
								mutex.Unlock()
								t.Logf("Player%d: PARTIDA FINALIZADA", userNum)
							default:
								t.Logf("Player%d: Mensagem não relacionada: %s", userNum, response.Type)
							}
							
							// Se é meu turno e a partida começou, faz uma jogada
							if myTurn && matchFound && !gameOver {
								// Escolhe uma carta aleatória (simulação)
								cardTypes := []string{"HYDRA", "QUIMERA", "GORGONA"}
								chosenCard := cardTypes[userNum%3] // Distribui cartas de forma previsível
								
								t.Logf("Player%d: FAZENDO JOGADA - Carta: %s", userNum, chosenCard)
								
								// Cria e envia jogada
								cardMoveMsg, err := protocol.CreateCardMove(userID, matchID, chosenCard)
								if err != nil {
									t.Logf("Player%d: ERRO ao criar mensagem de jogada: %v", userNum, err)
									continue
								}
								
								cardMoveMsg = append(cardMoveMsg, '\n')
								if _, err := conn.Write(cardMoveMsg); err != nil {
									t.Logf("Player%d: ERRO ao enviar jogada: %v", userNum, err)
									continue
								}
								
								mutex.Lock()
								jogadasRealizadas++
								mutex.Unlock()
								
								myTurn = false // Assume que não é mais meu turno após jogar
							}
						} else {
							if scanner.Err() != nil {
								t.Logf("Player%d: Erro no scanner: %v", userNum, scanner.Err())
								return
							}
							time.Sleep(100 * time.Millisecond)
						}
					case <-time.After(1 * time.Second):
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
	t.Logf("      TESTE STRESS - MATCHMAKING COMPLETO")
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