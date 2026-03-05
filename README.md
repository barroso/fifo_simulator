# Simulador de Fila FIFO

Projeto didático: simulador de fila de processamento com **todos os serviços rodando no Docker**. Frontend React para configurar testes e visualizar métricas em tempo real, backend Go consumindo Kafka, Dead Letter Queue (DLQ) e delay artificial.

## Pré-requisitos

- [Docker](https://docs.docker.com/get-docker/) e [Docker Compose](https://docs.docker.com/compose/install/)

## Como rodar (tudo no Docker)

**Importante:** Nenhum serviço deve rodar localmente. Kafka, API Go e Frontend devem rodar **somente** em containers (via `docker compose`). Não use `go run`, `npm run dev` ou Kafka local para executar a aplicação.

**Todos os serviços rodam em containers.** Não é necessário instalar Go, Node ou Kafka localmente para rodar o projeto.

```bash
# Na raiz do projeto
docker compose up --build
```

- **Frontend:** http://localhost:3000  
- **API (via frontend):** as chamadas `/api/*` são feitas pelo navegador para o mesmo host; o container do frontend faz proxy para o server.

Para subir também os serviços opcionais de monitoramento (Kafka UI, Prometheus, cAdvisor, Grafana):

```bash
docker compose --profile monitoring up --build
```

- Kafka UI: http://localhost:8081  
- Prometheus: http://localhost:9090  
- cAdvisor: http://localhost:8082  
- Grafana: http://localhost:3001 (login: `admin` / `admin`)

## Serviços no Docker

| Serviço   | Container   | Descrição                                      |
|-----------|-------------|------------------------------------------------|
| Kafka     | fifo-kafka  | Broker Kafka (KRaft), tópicos principal e DLQ |
| Server    | fifo-server | API Go + consumer Kafka + processamento       |
| Frontend  | fifo-frontend | React (build) servido por nginx + proxy /api |

A comunicação entre eles é interna na rede Docker (`kafka:9092`, `server:8080`). O usuário acessa apenas a porta 3000 (frontend), que repassa as requisições da API para o server.

## Uso rápido

1. Abra http://localhost:3000  
2. Em **Configuração**, defina quantidade de registros, tipo de tarefa (e-mail leve / imagem pesada), intervalo e % de falha. Clique em **Iniciar teste**.  
3. Em **Dashboard**, acompanhe em tempo real: tamanho da fila, latência, processados, falhas e DLQ.  
4. Em **DLQ**, veja as mensagens que falharam após o máximo de tentativas.  
5. Em **Análise**, veja gráficos de fila, latência e throughput.

## Componentes didáticos

- **Dashboard de métricas:** fila e latência em tempo real (SSE).  
- **Delay artificial:** tarefas “imagem” demoram mais que “e-mail” no consumer.  
- **Dead Letter Queue:** mensagens que falham repetidamente vão para a DLQ e aparecem na tela DLQ.  
- **Docker:** todos os serviços em containers; com o perfil `monitoring`, dá para ver uso de CPU/memória por container (Grafana + cAdvisor).
