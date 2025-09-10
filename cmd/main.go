package main

import (
	"top-card/internal/server"
	"top-card/internal/client"
	"fmt"
	"os"
)

func main(){
	mode := os.Getenv("MODE")

	switch mode{
	case "server":
		fmt.Println("Iniciando servidor TOP CARD")
		server.Run()
	case "client":
		fmt.Println("Iniciando cliente TOP CARD")
		client.Run()
	default:
		fmt.Println("Defina a variável MODE com 'server' ou 'client'")
	}
}