package test

import (
	"bufio"
	"fmt"
	"net"
	"os"
	"sync"
	"testing"
	"time"
	"top-card/internal/protocol"
)

// getServerAddr retorna o endereço do servidor a partir da variável de ambiente
func getServerAddr() string {
	serverAddr := os.Getenv("SERVER_ADDR")
	if serverAddr == "" {
		serverAddr = "localhost:8080" // fallback para desenvolvimento local
	}
	return serverAddr
}

// Teste de stress para abertura de pacotes
func TestStressCardPacks(t *testing.T) {
	serverAddr := getServerAddr()
	t.Logf("Conectando ao servidor: %s", serverAddr)
	
	numUsers := 100
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
			
			conn, err := net.DialTimeout("tcp", serverAddr, 5*time.Second)
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

// Teste de stress para login
func TestStressLogin(t *testing.T) {
	serverAddr := getServerAddr()
	t.Logf("Conectando ao servidor: %s", serverAddr)
	
	numUsers := 200
	var wg sync.WaitGroup
	
	loginSucesso := 0
	loginFalha := 0
	erroConexao := 0
	erroRegistro := 0
	loginDuplicado := 0
	var mutex sync.Mutex

	startTime := time.Now()

	for i := 0; i < numUsers; i++ {
		wg.Add(1)
		go func(userNum int) {
			defer wg.Done()
			
			username := fmt.Sprintf("login_user_%d_%d", userNum, time.Now().UnixNano())
			
			conn, err := net.DialTimeout("tcp", serverAddr, 5*time.Second)
			if err != nil {
				mutex.Lock()
				erroConexao++
				mutex.Unlock()
				t.Logf("User%d: ERRO CONEXAO - %v", userNum, err)
				return
			}
			defer conn.Close()
			
			conn.SetDeadline(time.Now().Add(10 * time.Second))
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
			
			// 2. LOGIN
			loginMsg, _ := protocol.CreateLoginRequest(username, "pass123")
			loginMsg = append(loginMsg, '\n')
			if _, err = conn.Write(loginMsg); err != nil {
				mutex.Lock()
				loginFalha++
				mutex.Unlock()
				return
			}
			
			if !scanner.Scan() {
				mutex.Lock()
				loginFalha++
				mutex.Unlock()
				return
			}
			
			response, err = protocol.DecodeMessage(scanner.Bytes())
			if err != nil {
				mutex.Lock()
				loginFalha++
				mutex.Unlock()
				return
			}
			
			loginResp, err := protocol.ExtractLoginResponse(response)
			if err != nil {
				mutex.Lock()
				loginFalha++
				mutex.Unlock()
				return
			}
			
			mutex.Lock()
			if loginResp.Success {
				loginSucesso++
				t.Logf("User%d: LOGIN SUCESSO - ID: %d", userNum, loginResp.UserID)
				
				// 3. TENTA LOGIN DUPLICADO (deve falhar)
				mutex.Unlock()
				
				// Abre segunda conexão para testar login duplicado
				conn2, err := net.DialTimeout("tcp", serverAddr, 5*time.Second)
				if err == nil {
					defer conn2.Close()
					conn2.SetDeadline(time.Now().Add(5 * time.Second))
					scanner2 := bufio.NewScanner(conn2)
					
					loginMsg2, _ := protocol.CreateLoginRequest(username, "pass123")
					loginMsg2 = append(loginMsg2, '\n')
					if _, err = conn2.Write(loginMsg2); err == nil {
						if scanner2.Scan() {
							response2, err := protocol.DecodeMessage(scanner2.Bytes())
							if err == nil {
								loginResp2, err := protocol.ExtractLoginResponse(response2)
								if err == nil && !loginResp2.Success {
									mutex.Lock()
									loginDuplicado++
									t.Logf("User%d: LOGIN DUPLICADO BLOQUEADO - %s", userNum, loginResp2.Message)
									mutex.Unlock()
								}
							}
						}
					}
				}
				
			} else {
				loginFalha++
				t.Logf("User%d: LOGIN FALHOU - %s", userNum, loginResp.Message)
				mutex.Unlock()
			}
			
		}(i+1)
		
		time.Sleep(100 * time.Millisecond)
	}
	
	wg.Wait()
	elapsed := time.Since(startTime)
	
	mutex.Lock()
	t.Logf("\n========================================")
	t.Logf("       TESTE STRESS - LOGIN")
	t.Logf("========================================")
	t.Logf("Usuarios testados: %d", numUsers)
	t.Logf("Tempo total: %v", elapsed)
	t.Logf("----------------------------------------")
	t.Logf("LOGIN SUCESSO:       %d", loginSucesso)
	t.Logf("LOGIN FALHA:         %d", loginFalha)
	t.Logf("LOGIN DUPLICADO:     %d", loginDuplicado)
	t.Logf("ERRO CONEXAO:        %d", erroConexao)
	t.Logf("ERRO REGISTRO:       %d", erroRegistro)
	t.Logf("----------------------------------------")
	total := loginSucesso + loginFalha + erroConexao + erroRegistro
	t.Logf("TOTAL:               %d", total)
	
	if loginSucesso > 0 {
		successRate := float64(loginSucesso) / float64(numUsers) * 100
		t.Logf("TAXA SUCESSO:        %.1f%%", successRate)
	}
	
	t.Logf("========================================")
	mutex.Unlock()
}

// Teste de stress para entrar na fila
func TestStressQueue(t *testing.T) {
	serverAddr := getServerAddr()
	t.Logf("Conectando ao servidor: %s", serverAddr)
	
	numUsers := 200
	var wg sync.WaitGroup
	
	filaSucesso := 0
	filaFalha := 0
	erroConexao := 0
	erroRegistro := 0
	erroLogin := 0
	erroPacote := 0
	filaDuplicada := 0
	filaSemCartas := 0
	var mutex sync.Mutex

	startTime := time.Now()

	for i := 0; i < numUsers; i++ {
		wg.Add(1)
		go func(userNum int) {
			defer wg.Done()
			
			username := fmt.Sprintf("queue_user_%d_%d", userNum, time.Now().UnixNano())
			
			conn, err := net.DialTimeout("tcp", serverAddr, 5*time.Second)
			if err != nil {
				mutex.Lock()
				erroConexao++
				mutex.Unlock()
				t.Logf("User%d: ERRO CONEXAO - %v", userNum, err)
				return
			}
			defer conn.Close()
			
			conn.SetDeadline(time.Now().Add(20 * time.Second))
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
				return
			}
			
			// 3. TENTA ENTRAR NA FILA SEM CARTAS (deve falhar)
			queueMsg1, _ := protocol.CreateQueueRequest(userID)
			queueMsg1 = append(queueMsg1, '\n')
			if _, err = conn.Write(queueMsg1); err != nil {
				mutex.Lock()
				filaFalha++
				mutex.Unlock()
				return
			}
			
			if !scanner.Scan() {
				mutex.Lock()
				filaFalha++
				mutex.Unlock()
				return
			}
			
			response, err = protocol.DecodeMessage(scanner.Bytes())
			if err != nil {
				mutex.Lock()
				filaFalha++
				mutex.Unlock()
				return
			}
			
			queueResp1, err := protocol.ExtractQueueResponse(response)
			if err != nil {
				mutex.Lock()
				filaFalha++
				mutex.Unlock()
				return
			}
			
			if !queueResp1.Success {
				mutex.Lock()
				filaSemCartas++
				t.Logf("User%d: FILA NEGADA SEM CARTAS - %s", userNum, queueResp1.Message)
				mutex.Unlock()
			}
			
			// 4. ABRIR PACOTE PARA TER CARTAS
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
				return
			}
			
			// 5. ENTRAR NA FILA COM CARTAS (deve funcionar)
			queueMsg2, _ := protocol.CreateQueueRequest(userID)
			queueMsg2 = append(queueMsg2, '\n')
			if _, err = conn.Write(queueMsg2); err != nil {
				mutex.Lock()
				filaFalha++
				mutex.Unlock()
				return
			}
			
			if !scanner.Scan() {
				mutex.Lock()
				filaFalha++
				mutex.Unlock()
				return
			}
			
			response, err = protocol.DecodeMessage(scanner.Bytes())
			if err != nil {
				mutex.Lock()
				filaFalha++
				mutex.Unlock()
				return
			}
			
			queueResp2, err := protocol.ExtractQueueResponse(response)
			if err != nil {
				mutex.Lock()
				filaFalha++
				mutex.Unlock()
				return
			}
			
			mutex.Lock()
			if queueResp2.Success {
				filaSucesso++
				t.Logf("User%d: FILA SUCESSO - tamanho: %d", userNum, queueResp2.QueueSize)
				mutex.Unlock()
				
				// 6. TENTA ENTRAR NA FILA NOVAMENTE (deve falhar - duplicado)
				queueMsg3, _ := protocol.CreateQueueRequest(userID)
				queueMsg3 = append(queueMsg3, '\n')
				if _, err = conn.Write(queueMsg3); err == nil {
					if scanner.Scan() {
						response, err := protocol.DecodeMessage(scanner.Bytes())
						if err == nil {
							queueResp3, err := protocol.ExtractQueueResponse(response)
							if err == nil && !queueResp3.Success {
								mutex.Lock()
								filaDuplicada++
								t.Logf("User%d: FILA DUPLICADA BLOQUEADA - %s", userNum, queueResp3.Message)
								mutex.Unlock()
							}
						}
					}
				}
				
			} else {
				filaFalha++
				t.Logf("User%d: FILA FALHOU - %s", userNum, queueResp2.Message)
				mutex.Unlock()
			}
			
		}(i+1)
		
		time.Sleep(150 * time.Millisecond)
	}
	
	wg.Wait()
	elapsed := time.Since(startTime)
	
	mutex.Lock()
	t.Logf("\n========================================")
	t.Logf("       TESTE STRESS - FILA")
	t.Logf("========================================")
	t.Logf("Usuarios testados: %d", numUsers)
	t.Logf("Tempo total: %v", elapsed)
	t.Logf("----------------------------------------")
	t.Logf("FILA SUCESSO:        %d", filaSucesso)
	t.Logf("FILA FALHA:          %d", filaFalha)
	t.Logf("FILA SEM CARTAS:     %d", filaSemCartas)
	t.Logf("FILA DUPLICADA:      %d", filaDuplicada)
	t.Logf("ERRO CONEXAO:        %d", erroConexao)
	t.Logf("ERRO REGISTRO:       %d", erroRegistro)
	t.Logf("ERRO LOGIN:          %d", erroLogin)
	t.Logf("ERRO PACOTE:         %d", erroPacote)
	t.Logf("----------------------------------------")
	total := filaSucesso + filaFalha + erroConexao + erroRegistro + erroLogin + erroPacote
	t.Logf("TOTAL:               %d", total)
	
	if filaSucesso > 0 {
		successRate := float64(filaSucesso) / float64(numUsers) * 100
		t.Logf("TAXA SUCESSO:        %.1f%%", successRate)
	}
	
	t.Logf("========================================")
	mutex.Unlock()
}