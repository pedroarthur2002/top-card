# TOP CARD

### Executar com Docker

- Fazer o *build*:
``` powershell
docker-compose build
```

- Executar o servidor:
 ``` powershell
docker-compose server
```

- Executar o *client*
``` powershell
docker-compose run --rm client
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
