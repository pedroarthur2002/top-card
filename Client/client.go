package main

import (
	"bufio"
	"fmt"
	"net"
	"os"
	"strings"
)

func main() {
    // Cliente com conexão no servidor na porta 8080
    conn, err := net.Dial("tcp", ":8080")
    if err != nil {
        fmt.Println("Falha na conexão com servidor! Erro: ", err)
        return
    }
    defer conn.Close()

    // Leitor para receber mensagens do servidor
    serverReader := bufio.NewReader(conn)
    // Reader para ler do teclado
    inputReader := bufio.NewReader(os.Stdin)

    for {
        // Lê mensagens do servidor até encontrar o prompt "> "
        for {
            msg, err := serverReader.ReadString('>')
            if err != nil {
                fmt.Println("Erro ao ler do servidor: ", err)
                return
            }
            
            fmt.Print(msg)
            
            // Se encontrou o prompt, para de ler e espera input
            if strings.HasSuffix(msg, ">") {
                break
            }
        }

        // Lê a entrada do usuário
        text, _ := inputReader.ReadString('\n')
        text = strings.TrimSpace(text)
        
        if text == "" {
            continue
        }

        // Envia a escolha para o servidor
        conn.Write([]byte(text + "\n"))
    }
}
