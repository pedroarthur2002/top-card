package main

import (
	"bufio"
	"fmt"
	"net"
)

func main(){
	// Criação do servidor (ouvindo na porta 8080)
	ln, err := net.Listen("tcp", ":8080")
	if err != nil{
		fmt.Println("Erro do tipo: ", err)
	}
	defer ln.Close()

	fmt.Println("Servidor rodando...")

	for {
		conn, err := ln.Accept()
		if err != nil{
			fmt.Println("Erro do tipo: ", err)
			continue
		}
		go handleConnection(conn)
	}
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
        
        choice := scanner.Text() // opção escolhida pelo cliente
        
        switch choice {
        case "1":
            conn.Write([]byte("Você escolheu fazer login\n"))
            // chamar a função de login
        case "2":
            conn.Write([]byte("Você escolheu criar seu jogador. Bem vindo!\n"))
            // Chamar a função de registrar
        case "3":
            conn.Write([]byte("Você escolheu sair. Até a próxima!\n"))
			// 
        default:
            conn.Write([]byte("Opção inválida. Tente novamente.\n"))
        }
    }
}

// Tela de login
func loginScreen(conn net.Conn) {
    menu := "1 - Fazer login\n2 - Criar jogador\n3 - Sair do jogo\n> "
    conn.Write([]byte(menu))
}