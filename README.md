# TOP CARD

### Executar com Docker

- Criar rede Docker
``` bash
docker network create topcard-network
```

- Fazer o *build*:
``` bash
docker build -t topcard-app .
```

- Rodar o servidor
 ``` bash
docker run -d --name topcard-server --network topcard-network -e MODE=server -p 8080:8080 topcard-app
```

- Rodar o *client*
``` bash
docker run -it --name topcard-client --network topcard-network -e MODE=client -e SERVER_ADDR=topcard-server:8080 topcard-app
```

### Executar localmente (no powershell)
Rodar o servidor
``` powershell
  $env:MODE="server"; go run main.go
```

Rodar o client 
``` powershell
  $env:MODE="client"; $env:SERVER_ADDR="127.0.0.1:8080"; go run main.go
```
