# TOP CARD

Este é um jogo de cartas 1v1 para a disciplina **TEC502 - Concorrência e Conectividade**. O sistema utiliza conceitos de programação concorrente, comunicação em rede e sincronização de estado entre clientes conectados.

## Tecnologias

- **Go**
- **Docker** e **Docker Compose**

## Como Executar

1. Clone o repositório:

```bash
git clone https://github.com/pedroarthur2002/top-card
cd top-card
```

2. Faça o *build* da imagem:

``` bash
docker-compose build
```

3. Execute o servidor:

``` bash
docker-compose up server
```

4. Execute o cliente:

``` bash
docker-compose run --rm client
```

### Execução distribuída

Caso queira executar o cliente numa máquina e os clientes em diferentes máquinas:

1. Descubra o IP local:

- Windows (É o endereço IPv4)

``` powershell
ipconfig
```

- Linux
``` bash
hostname -I
```

2. Execute o servidor: 

``` bash
docker-compose up server
```

3. Execute o cliente

``` bash
docker-compose run --rm -e SERVER_ADDR=192.168.1.102:8080 client
```

> Substitua o IP `192.168.1.102` pelo IP da máquina onde o servidor está rodando

## Estrutura do projeto

```
.
├── cmd/                # Aplicação principal
├── internal/           # Código privado da aplicação
├── test/               # Testes  
├── docker-compose.yml
├── Dockerfile
├── go.mod
├── go.sum
└── README.md
```

##  Testes

Para executar os testes é necessário:

1. Faça o *build* da imagem:

``` bash
docker-compose build
```

2. Execute o servidor:

``` bash
docker-compose up server
```

3. Execute o comando de testes:

- Teste de abertura dos pacotes:

``` bash
docker-compose --profile testing run --rm test go test ./test -run TestStressCardPacks -v
```

- Teste do matchmaking:

``` bash
docker-compose --profile testing run --rm test go test ./test -run TestStressMatchmaking -v
```

### Execução distribuída
Para executar os testes com servidor e teste em computadores diferentes é necessário:

1. Execute o servidor:

``` bash
docker-compose up server
```

2. Execute o teste de abertura dos pacotes: 

``` bash
SERVER_ADDR="192.168.1.100:8080" go test ./test -run TestStressCardPacks -v
```

3. Execute o teste de login's simultâneos: 

``` bash
SERVER_ADDR="192.168.1.100:8080" go test ./test -run TestStressLogin -v
```

2. Execute o teste de entrada na fila: 

``` bash
SERVER_ADDR="192.168.1.100:8080" go test ./test -run TestStressQueue -v
```

> Substitua o IP `192.168.1.102` pelo IP da máquina onde o servidor está rodando