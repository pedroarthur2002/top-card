package client

import (
	"bufio"
	"net"
	//"encoding/json"
	"fmt"
	"os"
	"strings"
)

func Run() {

	serverAddr := os.Getenv("SERVER_ADDR")

	conn, err := net.Dial("tcp", serverAddr)

	if err != nil {
		fmt.Println("Erro ao conectar no servidor:", err)
		return
	}
	defer conn.Close()

	fmt.Println("Conectado ao servidor TOP CARD!")

	reader := bufio.NewReader(os.Stdin)
	
	for {
		fmt.Println("\n========================")
		fmt.Println("Bem vindo ao TOP CARD!")
		fmt.Println("========================")
		fmt.Println("1 - Fazer login")
		fmt.Println("2 - Cadastrar-se")
		fmt.Println("3 - Abrir pacote de cartas")
		fmt.Println("4 - Buscar partida")
		fmt.Println("5 - Sair")
		
		fmt.Print("Insira sua opção: ")
		var choice int
		fmt.Scanf("%d", &choice)
		reader.ReadString('\n') // Limpa o buffer depois do scanf

		switch choice {
		case 1:
			handleLogin(conn, reader);

			// Implementar a lógica de mandar os dados para o servidor
			
		case 2:
			handleRegister(conn, reader);	
		
			// Implementar a lógica de mandar para o servidor
			
		case 5:
			fmt.Println("Você escolheu sair. Saindo...")
			return
			
		default:
			fmt.Println("Opção inválida!")
			// Limpa o buffer em caso de entrada inválida
			reader.ReadString('\n')
		}
	}
}

func handleLogin(conn net.Conn, reader *bufio.Reader){
	fmt.Println("\n--- LOGIN ---")
	fmt.Print("Insira seu nome de usuário: ")
	userName, _ := reader.ReadString('\n')
	userName = strings.TrimSpace(userName)

	fmt.Print("Digite sua senha: ")
	password, _ := reader.ReadString('\n')
	password = strings.TrimSpace(password)

	// Teste das entradas, retirar depois
	fmt.Printf("\nNovo usuário: %s\n", userName)
	fmt.Printf("Senha: %s\n", password)
	fmt.Println("Cadastro testado com sucesso!")
}

func handleRegister(cont net.Conn, reader *bufio.Reader){
	fmt.Println("\n--- CADASTRO ---")
	fmt.Print("Insira um nome de usuário: ")
	userName, _ := reader.ReadString('\n')
	userName = strings.TrimSpace(userName)

	fmt.Print("Digite sua senha: ")
	password, _ := reader.ReadString('\n')
	password = strings.TrimSpace(password)

	// Teste das entradas, retirar depois
	fmt.Printf("\nNovo usuário: %s\n", userName)
	fmt.Printf("Senha: %s\n", password)
	fmt.Println("Cadastro testado com sucesso!")
}