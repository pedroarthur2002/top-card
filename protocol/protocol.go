package protocol

import "encoding/json"

// Tipos de mensagens do protocolo
const (
	MSG_LOGIN_REQUEST    = "LOGIN_REQUEST"
	MSG_LOGIN_RESPONSE   = "LOGIN_RESPONSE"
	MSG_REGISTER_REQUEST = "REGISTER_REQUEST"
	MSG_REGISTER_RESPONSE = "REGISTER_RESPONSE"
)

// Estrutura base para todas as mensagens
type Message struct {
	Type string      `json:"type"`
	Data interface{} `json:"data"`
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

// Função para decodificar mensagem recebida
func DecodeMessage(data []byte) (*Message, error) {
	var message Message
	err := json.Unmarshal(data, &message)
	if err != nil {
		return nil, err
	}
	return &message, nil
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