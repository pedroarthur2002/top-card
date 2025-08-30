# TOP CARD

### Executar com Docker
- Fazer o *build*:
``` bash
docker build -t top-card .
```
- Rodar o servidor
 ``` bash
docker run --rm -e MODE=server top-card
```
- Rodar o *client*
``` bash
docker run --rm -e MODE=client -e SERVER_ADDR=127.0.0.1:8080 top-card
```
### Executar localmente
Rodar o servidor
``` powershell
  $env:MODE="server"; go run main.go
```
Rodar o client 
``` powershell
  $env:MODE="client"; $env:SERVER_ADDR="127.0.0.1:8080"; go run main.go
```
