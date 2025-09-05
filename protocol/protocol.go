package protocol

import (
	"encoding/json"
)

// Tipos de mensagens do protocolo
const (
	MSG_LOGIN_REQUEST    = "LOGIN_REQUEST"
	MSG_PING_REQUEST     = "PING_REQUEST"
	MSG_PING_RESPONSE    = "PING_RESPONSE" 
	MSG_LOGIN_RESPONSE   = "LOGIN_RESPONSE"
	MSG_REGISTER_REQUEST = "REGISTER_REQUEST"
	MSG_REGISTER_RESPONSE = "REGISTER_RESPONSE"
	MSG_QUEUE_REQUEST    = "QUEUE_REQUEST"
	MSG_QUEUE_RESPONSE   = "QUEUE_RESPONSE"
	MSG_MATCH_FOUND      = "MATCH_FOUND"
	MSG_MATCH_START      = "MATCH_START"
	MSG_MATCH_END        = "MATCH_END"
	MSG_GAME_MOVE     = "GAME_MOVE"
	MSG_GAME_STATE    = "GAME_STATE" 
	MSG_TURN_UPDATE   = "TURN_UPDATE"
	MSG_STATS_REQUEST  = "STATS_REQUEST"
	MSG_STATS_RESPONSE = "STATS_RESPONSE"
)

// Estrutura base para todas as mensagens
type Message struct {
	Type string      `json:"type"`
	Data interface{} `json:"data"`
}

// Estrutura para resposta de estatísticas
type StatsResponse struct {
	Success   bool    `json:"success"`
	Message   string  `json:"message"`
	UserName  string  `json:"username,omitempty"`
	Wins      int     `json:"wins,omitempty"`
	Losses    int     `json:"losses,omitempty"`
	WinRate   float64 `json:"win_rate,omitempty"`
}

// Estrutura para requisição de estatísticas
type StatsRequest struct {
	UserID int `json:"user_id"`
}

// Estrutura para requisição de ping
type PingRequest struct {
	UserID    int   `json:"user_id"`
}

// Estrutura para resposta de ping
type PingResponse struct { 
	Success   bool  `json:"success"`  
	Message   string `json:"message"`
}

// Estrutura para requisição de login
type LoginRequest struct {
	UserName string `json:"username"`
	Password string `json:"password"`
}

// Estrutura para resposta de login
type LoginResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
	UserID  int    `json:"user_id,omitempty"` // Apenas se login bem-sucedido
}

// Estrutura para requisição de cadastro
type RegisterRequest struct {
	UserName string `json:"username"`
	Password string `json:"password"`
}

// Estrutura para resposta de cadastro
type RegisterResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
	UserID  int    `json:"user_id,omitempty"` // Apenas se cadastro bem-sucedido
}

// Estrutura para requisição de fila
type QueueRequest struct {
	UserID int `json:"user_id"`
}

// Estrutura para resposta de fila
type QueueResponse struct {
	Success    bool   `json:"success"`
	Message    string `json:"message"`
	QueueSize  int    `json:"queue_size"`
}

// Estrutura para notificação de partida encontrada
type MatchFound struct {
	MatchID      int    `json:"match_id"`
	OpponentID   int    `json:"opponent_id"`
	OpponentName string `json:"opponent_name"`
	Message      string `json:"message"`
}

// Estrutura para início de partida
type MatchStart struct {
	MatchID int    `json:"match_id"`
	Message string `json:"message"`
}

// Estrutura para fim de partida
type MatchEnd struct {
	MatchID    int    `json:"match_id"`
	WinnerID   int    `json:"winner_id"`
	WinnerName string `json:"winner_name"`
	Message    string `json:"message"`
}

// Estrutura para jogada do jogo
type GameMove struct {
	UserID  int `json:"user_id"`
	MatchID int `json:"match_id"`
	Number  int `json:"number"`
}

// Estrutura para estado do jogo
type GameState struct {
	MatchID       int    `json:"match_id"`
	Message       string `json:"message"`
	YourTurn      bool   `json:"your_turn"`
	OpponentMoved bool   `json:"opponent_moved"`
	GameOver      bool   `json:"game_over"`
}

// Estrutura para atualização de turno
type TurnUpdate struct {
	MatchID  int    `json:"match_id"`
	Message  string `json:"message"`
	YourTurn bool   `json:"your_turn"`
}

// Função para criar mensagem de requisição de estatísticas
func CreateStatsRequest(userID int) ([]byte, error) {
	statsReq := StatsRequest{
		UserID: userID,
	}

	message := Message{
		Type: MSG_STATS_REQUEST,
		Data: statsReq,
	}

	return json.Marshal(message)
}

// Função para criar mensagem de resposta de estatísticas
func CreateStatsResponse(success bool, message, userName string, wins, losses int, winRate float64) ([]byte, error) {
	statsResp := StatsResponse{
		Success:  success,
		Message:  message,
		UserName: userName,
		Wins:     wins,
		Losses:   losses,
		WinRate:  winRate,
	}

	msg := Message{
		Type: MSG_STATS_RESPONSE,
		Data: statsResp,
	}

	return json.Marshal(msg)
}

// Função para criar mensagem de requisição de ping
func CreatePingRequest(userID int) ([]byte, error) {
	pingReq := PingRequest{
		UserID:    userID,
	}

	message := Message{
		Type: MSG_PING_REQUEST,
		Data: pingReq,
	}

	return json.Marshal(message)
}

// Função para criar mensagem de reposta de ping
func CreatePingResponse(success bool, message string) ([]byte, error) {
	pingResponse := PingResponse{ 
		Success:   success,
		Message:   message,
	}

	msg := Message{
		Type: MSG_PING_RESPONSE,
		Data: pingResponse,
	}

	return json.Marshal(msg)
}

// Função para criar mensagem de requisição de login
func CreateLoginRequest(userName, password string) ([]byte, error) {
	loginReq := LoginRequest{
		UserName: userName,
		Password: password,
	}
	
	message := Message{
		Type: MSG_LOGIN_REQUEST,
		Data: loginReq,
	}
	
	return json.Marshal(message)
}

// Função para criar mensagem de resposta de login
func CreateLoginResponse(success bool, message string, userID int) ([]byte, error) {
	loginResp := LoginResponse{
		Success: success,
		Message: message,
		UserID:  userID,
	}
	
	msg := Message{
		Type: MSG_LOGIN_RESPONSE,
		Data: loginResp,
	}
	
	return json.Marshal(msg)
}

// Função para criar mensagem de requisição de cadastro
func CreateRegisterRequest(userName, password string) ([]byte, error) {
	registerReq := RegisterRequest{
		UserName: userName,
		Password: password,
	}
	
	message := Message{
		Type: MSG_REGISTER_REQUEST,
		Data: registerReq,
	}
	
	return json.Marshal(message)
}

// Função para criar mensagem de resposta de cadastro
func CreateRegisterResponse(success bool, message string, userID int) ([]byte, error) {
	registerResp := RegisterResponse{
		Success: success,
		Message: message,
		UserID:  userID,
	}
	
	msg := Message{
		Type: MSG_REGISTER_RESPONSE,
		Data: registerResp,
	}
	
	return json.Marshal(msg)
}

// Função para criar mensagem de requisição de fila
func CreateQueueRequest(userID int) ([]byte, error) {
	queueReq := QueueRequest{
		UserID: userID,
	}
	
	message := Message{
		Type: MSG_QUEUE_REQUEST,
		Data: queueReq,
	}
	
	return json.Marshal(message)
}

// Função para criar mensagem de resposta de fila
func CreateQueueResponse(success bool, message string, queueSize int) ([]byte, error) {
	queueResp := QueueResponse{
		Success:   success,
		Message:   message,
		QueueSize: queueSize,
	}
	
	msg := Message{
		Type: MSG_QUEUE_RESPONSE,
		Data: queueResp,
	}
	
	return json.Marshal(msg)
}

// Função para criar mensagem de partida encontrada
func CreateMatchFound(matchID, opponentID int, opponentName, message string) ([]byte, error) {
	matchFound := MatchFound{
		MatchID:      matchID,
		OpponentID:   opponentID,
		OpponentName: opponentName,
		Message:      message,
	}
	
	msg := Message{
		Type: MSG_MATCH_FOUND,
		Data: matchFound,
	}
	
	return json.Marshal(msg)
}

// Função para criar mensagem de início de partida
func CreateMatchStart(matchID int, message string) ([]byte, error) {
	matchStart := MatchStart{
		MatchID: matchID,
		Message: message,
	}
	
	msg := Message{
		Type: MSG_MATCH_START,
		Data: matchStart,
	}
	
	return json.Marshal(msg)
}

// Função para criar mensagem de fim de partida
func CreateMatchEnd(matchID, winnerID int, winnerName, message string) ([]byte, error) {
	matchEnd := MatchEnd{
		MatchID:    matchID,
		WinnerID:   winnerID,
		WinnerName: winnerName,
		Message:    message,
	}
	
	msg := Message{
		Type: MSG_MATCH_END,
		Data: matchEnd,
	}
	
	return json.Marshal(msg)
}

// Função para decodificar mensagem recebida
func DecodeMessage(data []byte) (*Message, error) {
	var message Message
	err := json.Unmarshal(data, &message)
	if err != nil {
		return nil, err
	}
	return &message, nil
}

// Função para extrair dados de requisição de estatísticas
func ExtractStatsRequest(message *Message) (*StatsRequest, error) {
	dataBytes, err := json.Marshal(message.Data)
	if err != nil {
		return nil, err
	}

	var statsReq StatsRequest
	err = json.Unmarshal(dataBytes, &statsReq)
	if err != nil {
		return nil, err
	}

	return &statsReq, nil
}

// Função para extrair dados de resposta de estatísticas
func ExtractStatsResponse(message *Message) (*StatsResponse, error) {
	dataBytes, err := json.Marshal(message.Data)
	if err != nil {
		return nil, err
	}

	var statsResp StatsResponse
	err = json.Unmarshal(dataBytes, &statsResp)
	if err != nil {
		return nil, err
	}

	return &statsResp, nil
}

// Função para extrair dados de ping request
func ExtractPingRequest(message *Message) (*PingRequest, error) {
	dataBytes, err := json.Marshal(message.Data)
	if err != nil {
		return nil, err
	}
	
	var PingRequest PingRequest
	err = json.Unmarshal(dataBytes, &PingRequest)
	if err != nil {
		return nil, err
	}
	
	return &PingRequest, nil
}

// Função para extrair dados de ping response
func ExtractPingResponse(message *Message) (*PingResponse, error) { 
	dataBytes, err := json.Marshal(message.Data)
	if err != nil {
		return nil, err
	}
	
	var pingResponse PingResponse
	err = json.Unmarshal(dataBytes, &pingResponse)
	if err != nil {
		return nil, err
	}
	
	return &pingResponse, nil
}

// Função para extrair dados de login request
func ExtractLoginRequest(message *Message) (*LoginRequest, error) {
	dataBytes, err := json.Marshal(message.Data)
	if err != nil {
		return nil, err
	}
	
	var loginReq LoginRequest
	err = json.Unmarshal(dataBytes, &loginReq)
	if err != nil {
		return nil, err
	}
	
	return &loginReq, nil
}

// Função para extrair dados de login response
func ExtractLoginResponse(message *Message) (*LoginResponse, error) {
	dataBytes, err := json.Marshal(message.Data)
	if err != nil {
		return nil, err
	}
	
	var loginResp LoginResponse
	err = json.Unmarshal(dataBytes, &loginResp)
	if err != nil {
		return nil, err
	}
	
	return &loginResp, nil
}

// Função para extrair dados de cadastro request
func ExtractRegisterRequest(message *Message) (*RegisterRequest, error) {
	dataBytes, err := json.Marshal(message.Data)
	if err != nil {
		return nil, err
	}
	
	var registerReq RegisterRequest
	err = json.Unmarshal(dataBytes, &registerReq)
	if err != nil {
		return nil, err
	}
	
	return &registerReq, nil
}

// Função para extrair dados de cadastro response
func ExtractRegisterResponse(message *Message) (*RegisterResponse, error) {
	dataBytes, err := json.Marshal(message.Data)
	if err != nil {
		return nil, err
	}
	
	var registerResp RegisterResponse
	err = json.Unmarshal(dataBytes, &registerResp)
	if err != nil {
		return nil, err
	}
	
	return &registerResp, nil
}

// Função para extrair dados de fila request
func ExtractQueueRequest(message *Message) (*QueueRequest, error) {
	dataBytes, err := json.Marshal(message.Data)
	if err != nil {
		return nil, err
	}
	
	var queueReq QueueRequest
	err = json.Unmarshal(dataBytes, &queueReq)
	if err != nil {
		return nil, err
	}
	
	return &queueReq, nil
}

// Função para extrair dados de fila response
func ExtractQueueResponse(message *Message) (*QueueResponse, error) {
	dataBytes, err := json.Marshal(message.Data)
	if err != nil {
		return nil, err
	}
	
	var queueResp QueueResponse
	err = json.Unmarshal(dataBytes, &queueResp)
	if err != nil {
		return nil, err
	}
	
	return &queueResp, nil
}

// Função para extrair dados de partida encontrada
func ExtractMatchFound(message *Message) (*MatchFound, error) {
	dataBytes, err := json.Marshal(message.Data)
	if err != nil {
		return nil, err
	}
	
	var matchFound MatchFound
	err = json.Unmarshal(dataBytes, &matchFound)
	if err != nil {
		return nil, err
	}
	
	return &matchFound, nil
}

// Função para extrair dados de início de partida
func ExtractMatchStart(message *Message) (*MatchStart, error) {
	dataBytes, err := json.Marshal(message.Data)
	if err != nil {
		return nil, err
	}
	
	var matchStart MatchStart
	err = json.Unmarshal(dataBytes, &matchStart)
	if err != nil {
		return nil, err
	}
	
	return &matchStart, nil
}

// Função para extrair dados de fim de partida
func ExtractMatchEnd(message *Message) (*MatchEnd, error) {
	dataBytes, err := json.Marshal(message.Data)
	if err != nil {
		return nil, err
	}
	
	var matchEnd MatchEnd
	err = json.Unmarshal(dataBytes, &matchEnd)
	if err != nil {
		return nil, err
	}
	
	return &matchEnd, nil
}

// Função para criar mensagem de jogada
func CreateGameMove(userID, matchID, number int) ([]byte, error) {
	gameMove := GameMove{
		UserID:  userID,
		MatchID: matchID,
		Number:  number,
	}
	
	message := Message{
		Type: MSG_GAME_MOVE,
		Data: gameMove,
	}
	
	return json.Marshal(message)
}

// Função para criar mensagem de estado do jogo
func CreateGameState(matchID int, message string, yourTurn, opponentMoved, gameOver bool) ([]byte, error) {
	gameState := GameState{
		MatchID:       matchID,
		Message:       message,
		YourTurn:      yourTurn,
		OpponentMoved: opponentMoved,
		GameOver:      gameOver,
	}
	
	msg := Message{
		Type: MSG_GAME_STATE,
		Data: gameState,
	}
	
	return json.Marshal(msg)
}

// Função para criar mensagem de atualização de turno
func CreateTurnUpdate(matchID int, message string, yourTurn bool) ([]byte, error) {
	turnUpdate := TurnUpdate{
		MatchID:  matchID,
		Message:  message,
		YourTurn: yourTurn,
	}
	
	msg := Message{
		Type: MSG_TURN_UPDATE,
		Data: turnUpdate,
	}
	
	return json.Marshal(msg)
}

// Função para extrair dados de jogada
func ExtractGameMove(message *Message) (*GameMove, error) {
	dataBytes, err := json.Marshal(message.Data)
	if err != nil {
		return nil, err
	}
	
	var gameMove GameMove
	err = json.Unmarshal(dataBytes, &gameMove)
	if err != nil {
		return nil, err
	}
	
	return &gameMove, nil
}

// Função para extrair dados de estado do jogo
func ExtractGameState(message *Message) (*GameState, error) {
	dataBytes, err := json.Marshal(message.Data)
	if err != nil {
		return nil, err
	}
	
	var gameState GameState
	err = json.Unmarshal(dataBytes, &gameState)
	if err != nil {
		return nil, err
	}
	
	return &gameState, nil
}

// Função para extrair dados de atualização de turno
func ExtractTurnUpdate(message *Message) (*TurnUpdate, error) {
	dataBytes, err := json.Marshal(message.Data)
	if err != nil {
		return nil, err
	}
	
	var turnUpdate TurnUpdate
	err = json.Unmarshal(dataBytes, &turnUpdate)
	if err != nil {
		return nil, err
	}
	
	return &turnUpdate, nil
}