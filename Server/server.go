package main

import (
	"bufio"
	"fmt"
	"net"
	"strings"
	"top-card/Player"
)

var players []Player.Player
var nextID = 1

func main() {
	// Criação do servidor (ouvindo na porta 8080)
	ln, err := net.Listen("tcp", ":8080")
	if err != nil {
		fmt.Println("Erro do tipo: ", err)
		return
	}
	defer ln.Close()

	fmt.Println("Servidor rodando...")

	for {
		conn, err := ln.Accept()
		if err != nil {
			fmt.Println("Erro do tipo: ", err)
			continue
		}
		go handleConnection(conn)
	}
}

func usernameExists(username string) bool {
	for _, player := range players {
		if player.GetUserName() == username { 
			return true
		}
	}
	return false
}

// Registro do jogador
func registerPlayer(userName, password string) (Player.Player, error) {
	if usernameExists(userName) {
		return Player.Player{}, fmt.Errorf("nome de usuário já existe")
	}

	newPlayer := Player.NewPlayer(nextID, userName, password)
	players = append(players, newPlayer)
	nextID++

	return newPlayer, nil
}

func handleConnection(conn net.Conn) {
	defer conn.Close()

	conn.Write([]byte("Bem vindo ao jogo TOP CARD\n"))
	conn.Write([]byte("Será que tens o que é necessário para esmagares a minha rata?\n"))

	// Scanner para ler o que o cliente envia
	scanner := bufio.NewScanner(conn)

	for {
		// Mostra menu
		loginScreen(conn)

		// Espera o cliente digitar algo
		if !scanner.Scan() {
			fmt.Println("Cliente desconectou")
			return
		}

		choice := strings.TrimSpace(scanner.Text()) // opção escolhida pelo cliente

		switch choice {
		case "1":
			conn.Write([]byte("Você escolheu fazer login\n"))
			// Implementar login posteriormente

		case "2":
			conn.Write([]byte("Você escolheu criar seu jogador. Bem vindo!\n"))
			handleRegistration(conn, scanner)

		case "3":
			conn.Write([]byte("Você escolheu sair. Até a próxima!\n"))
			return

		default:
			conn.Write([]byte("Opção inválida. Tente novamente.\n"))
		}
	}
}

// Função para o registro
func handleRegistration(conn net.Conn, scanner *bufio.Scanner) {
	conn.Write([]byte("Digite um nome de usuário> "))
	if !scanner.Scan() {
		conn.Write([]byte("Erro ao ler o nome de usuário\n"))
		return
	}

	username := strings.TrimSpace(scanner.Text())

	conn.Write([]byte("Digite uma senha> "))
	if !scanner.Scan() {
		conn.Write([]byte("Erro ao ler senha\n"))
		return
	}

	password := strings.TrimSpace(scanner.Text())

	newPlayer, err := registerPlayer(username, password)
	if err != nil {
		conn.Write([]byte("Erro no registro: " + err.Error() + "\n"))
		return
	}

	conn.Write([]byte("Jogador criado com sucesso! ID: " + fmt.Sprint(newPlayer.GetID()) + "\n"))
	conn.Write([]byte("Bem vindo, " + newPlayer.GetUserName() + "!\n"))
}

// Tela de login
func loginScreen(conn net.Conn) {
	menu := "1 - Fazer login\n2 - Criar jogador\n3 - Sair do jogo\n>"
	conn.Write([]byte(menu))
}