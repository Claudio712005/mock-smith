# MockSmith

[![CI](https://github.com/Claudio712005/mock-smith/actions/workflows/ci.yml/badge.svg)](https://github.com/Claudio712005/mock-smith/actions/workflows/ci.yml)
[![Go 1.26+](https://img.shields.io/badge/go-1.26%2B-00ADD8?logo=go&logoColor=white)](https://go.dev/dl/)

Engine de mock de APIs em runtime, guiado por specs OpenAPI/Swagger.

O MockSmith lê um documento OpenAPI 3 (ou Swagger) e sobe um mock HTTP ao vivo
direto do contrato — sem arquivos de mock estáticos, zero config por padrão.
Aponte para uma spec e tenha respostas realistas na hora.

```bash
mocksmith run openapi.yaml
```

```txt
MockSmith running on :8080
Loaded 23 endpoints
Profile: happy
```

> Status: **Fase 6**. Carrega a spec, descobre os endpoints e serve respostas
> guiadas por um **profile de runtime** (`happy`, `sad`, `resilience`, `chaos`) —
> sucesso, erro de negócio, erro de servidor, timeout, corpo malformado e
> desconexão. Um **pipeline de interceptors** compõe comportamentos sobre o
> profile, configuráveis **por endpoint** via flags (`--slow`, `--fail`,
> `--timeout`, `--corrupt`), incluindo **sequências stateful** (`--sequence`).
> Uma **Admin API** muda o comportamento **em runtime, sem restart**
> (`mocksmith inject`). Tudo isso cabe num **arquivo `mocksmith.yaml`**
> (`run --config`). Dashboard no roadmap abaixo.

---

## Funcionalidades (atuais)

- **Carga de OpenAPI** — YAML ou JSON, OpenAPI 3 / Swagger; schema validado e
  `$ref`s resolvidos no carregamento.
- **Descoberta de endpoints** — cada path + método vira uma rota ativa. Templates
  OpenAPI como `/users/{id}` casam direto com o roteador.
- **Respostas realistas** — prioridade na geração do corpo:
  1. exemplo explícito do OpenAPI,
  2. exemplo do schema,
  3. dado falso conforme o formato (email, uuid, uri, date/date-time, enums, números, …).
- **Profiles de runtime** — o comportamento de cada requisição é sorteado por
  peso conforme o `--profile`: sucesso, erro de negócio (4xx documentado), erro
  de servidor (500/503), timeout (504 após atraso), corpo JSON malformado e
  desconexão da conexão. Veja [Profiles](#profiles).
- **Resposta de sucesso** — a documentada (menor 2xx), ou `204` quando nenhuma
  é documentada. Prioridade do corpo descrita acima.
- **Admin API em runtime** — sobrescreve status/latência por endpoint sem
  reiniciar, via `/__mocksmith/runtime` ou `mocksmith inject`. Veja
  [Admin API](#admin-api--overrides-em-runtime).

---

## Casos de uso

O ponto forte do MockSmith é **desbloquear o desenvolvimento quando o serviço do
qual você depende ainda não existe, está incompleto ou está fora do ar** — e
ainda permitir testar como o *seu* serviço reage a falhas dele.

### 1. O microsserviço de que você depende ainda não foi escrito

Os times de front e back combinam o contrato OpenAPI primeiro. Com a spec em mãos,
suba o mock e desenvolva contra ele **no mesmo dia**, sem esperar a implementação.

```bash
# time de pagamentos só entregou o contrato; você já consome
mocksmith run payments-api.yaml --addr :9000
curl -X POST localhost:9000/payments   # resposta realista, gerada do schema
```

Quando o serviço real ficar pronto, é só trocar a URL — o contrato é o mesmo.

### 2. A dependência existe, mas está instável ou indisponível

Em vez de travar quando a API de terceiros cai (ou tem rate limit no ambiente de
dev), aponte para o mock e siga trabalhando — inclusive reproduzindo a
instabilidade dela.

```bash
# 90% ok, 5% timeout, 5% 503 — como um serviço meia-boca de verdade
mocksmith run partner-api.yaml --profile resilience

# ou modele a falha exata que você viu: /auth cai 20% das vezes
mocksmith run partner-api.yaml --timeout /auth=20% --slow /auth=800ms
```

### 3. Testar a resiliência do SEU serviço (retry, circuit breaker, fallback)

Aponte seu serviço para o mock e force o downstream a falhar do jeito que você
quer. Valida retry, timeout, circuit breaker e telas de fallback **sem tocar no
código do downstream**.

```bash
# downstream sempre 503: seu circuit breaker deve abrir
mocksmith run inventory-api.yaml --fail /stock=503

# corpo corrompido em 10% das respostas: seu parser aguenta?
mocksmith run inventory-api.yaml --corrupt /stock=10%

# caos total: 1 em 5 requisições dá algo errado
mocksmith run inventory-api.yaml --profile chaos
```

### 4. Fluxos assíncronos e polling (job que processa e depois conclui)

Endpoints de status que respondem `202 Accepted` algumas vezes e depois `200 OK`
são chatos de mockar à mão. Com `--sequence` é uma linha.

```bash
# 1ª e 2ª chamada → 202 (processando), 3ª em diante → 200 (pronto)
mocksmith run jobs-api.yaml --sequence /jobs/{id}=202,202,200
```

### 5. Reproduzir um incidente de produção — sem redeploy

Aconteceu um `503` intermitente com latência alta num endpoint? Reproduza ao vivo,
ajuste e desfaça, **sem reiniciar o mock**, via Admin API.

```bash
mocksmith run api.yaml &                              # mock rodando
mocksmith inject "/payments:503(30%)" --latency-ms 2000   # injeta o incidente
# ... investiga o comportamento do cliente ...
mocksmith inject --clear                              # volta ao normal
```

### 6. Testes de integração no CI

Suba o mock como dependência determinística do pipeline: sem rede externa, sem
flakiness, comportamento controlado por flags.

```bash
mocksmith run api.yaml --addr :8080 &
go test ./...            # seus testes batem no mock, não na internet
```

> Dica: combine flags por endpoint para montar cenários ricos numa linha só —
> ex.: `--fail /payments=503 --slow /pets=1s --sequence /jobs=202,202,200`.

---

## Profiles

O `--profile` define a distribuição de comportamentos por requisição. Os pesos
são sorteados de forma independente a cada chamada.

| Profile      | Distribuição                                                              |
|--------------|--------------------------------------------------------------------------|
| `happy`      | 100% sucesso                                                              |
| `sad`        | 70% sucesso · 30% erro de negócio (4xx documentado, ou `400` genérico)    |
| `resilience` | 90% sucesso · 5% timeout (504) · 5% `503`                                 |
| `chaos`      | 80% sucesso · 5% JSON malformado · 5% timeout · 5% desconexão · 5% `500`  |

Comportamentos:

- **Erro de negócio** — devolve o menor erro `4xx` documentado do endpoint; sem
  nenhum, sintetiza um `400` com corpo JSON genérico (`{ "code", "message" }`).
- **Erro de servidor** — devolve o erro documentado com aquele status, ou um
  corpo genérico com o status pedido.
- **Timeout** — dorme ~30s (simulando serviço travado) e então responde `504`;
  se o cliente cancelar antes, nada é escrito.
- **Malformado** — responde `200` com `Content-Type: application/json` e um corpo
  JSON propositalmente inválido (testa parsing/resiliência do cliente).
- **Desconexão** — derruba a conexão sem responder (cliente vê erro de conexão).

```bash
mocksmith run examples/petstore.yaml --profile sad
mocksmith run examples/petstore.yaml --profile chaos
```

### Forçar um status (`--force-status`)

`--force-status <code>` força um status HTTP **apenas nos endpoints que o
documentam** na spec (como sucesso ou erro). Endpoints que **não** mapeiam aquele
status ignoram a flag e seguem o `--profile` normalmente.

Combina com qualquer profile. Exemplo — profile `happy` + `--force-status 500`:
todo endpoint com `500` mapeado responde `500`; os demais continuam no happy.

```bash
mocksmith run examples/petstore.yaml --profile happy --force-status 503
```

Com a [spec de exemplo](examples/petstore.yaml):

```bash
curl -s -o /dev/null -w "%{http_code}\n" -X POST localhost:8080/payments  # 503 (mapeado → forçado)
curl -s -o /dev/null -w "%{http_code}\n"      localhost:8080/pets         # 200 (sem 503 → happy)
curl -s -o /dev/null -w "%{http_code}\n"      localhost:8080/pets/1       # 200 (só tem 404 → happy)
```

Detalhes:

- Vale para qualquer status documentado, não só erros — forçar o status de
  sucesso (ex.: `201`) só devolve a resposta de sucesso.
- Erro forçado usa o corpo do erro documentado daquele status.
- `0` (padrão) desliga. Valores fora de `100–599` são rejeitados na inicialização.

---

## Interceptors

Além do profile, o MockSmith aplica um **pipeline de interceptors**
(Chain of Responsibility) a cada requisição, **antes** de escrever a resposta.
Cada interceptor pode introduzir efeitos (ex.: atraso) e/ou sobrescrever o
comportamento sorteado pelo profile. Compõem-se em ordem, sem acoplar o servidor
a cada um.

Interceptors disponíveis no engine:

| Interceptor             | Efeito                                                       |
|-------------------------|--------------------------------------------------------------|
| `LatencyInterceptor`    | Atrasa toda resposta por uma duração fixa.                   |
| `FailureInterceptor`    | Sobrescreve com erro de servidor numa fração das requisições.|
| `TimeoutInterceptor`    | Sobrescreve com timeout (504) numa fração das requisições.   |
| `CorruptionInterceptor` | Sobrescreve com corpo malformado numa fração das requisições.|

### Flags por endpoint

Quatro flags injetam interceptors. Todas são **repetíveis** e usam o formato
`[path=]valor`: com `path=`, o efeito vale só naquele endpoint (casamento exato
contra o template OpenAPI, ex.: `/pets/{petId}`); sem `path=`, vale para **todos**.
Endpoints fora do escopo seguem o profile normalmente.

| Flag        | Valor             | Efeito                                                        |
|-------------|-------------------|--------------------------------------------------------------|
| `--slow`    | duração (`2s`)    | Adiciona latência à resposta. Respeita o cancelamento do cliente. |
| `--fail`    | status (`503`)    | Força esse status HTTP (sempre).                             |
| `--timeout` | taxa (`20%`)      | Injeta timeout (504, após ~30s) nessa fração das requisições. |
| `--corrupt` | taxa (`10%`)      | Corrompe o corpo JSON nessa fração das requisições.          |

Taxa aceita `20%` ou `0.2`. Duração aceita qualquer formato Go (`500ms`, `2s`,
`1m`). Valores inválidos (duração ruim, status fora de `100–599`, taxa fora de
`0–100%`) são rejeitados na inicialização.

```bash
# 2s de latência só em /payments; o resto responde na hora
mocksmith run examples/petstore.yaml --slow /payments=2s

# compõe vários comportamentos, por endpoint
mocksmith run examples/petstore.yaml \
  --fail /payments=503 \
  --timeout /pets=20% \
  --corrupt /users/{id}=10%

# latência global (sem path) + falha pontual
mocksmith run examples/petstore.yaml --slow 300ms --fail /payments=503
```

### Sequências stateful (`--sequence`)

`--sequence [path=]s1,s2,...` devolve os status **em ordem, um por requisição** ao
endpoint. Esgotada a lista, **fixa no último** — ideal para polling (ex.: job que
fica `202 Accepted` e depois `200 OK`). É stateful por endpoint, com contador
seguro para concorrência.

```bash
# 1ª req → 202, 2ª → 202, 3ª em diante → 200
mocksmith run examples/petstore.yaml --sequence /jobs=202,202,200
```

O status escolhido é renderizado como a resposta da spec: igual ao sucesso
documentado do endpoint usa o corpo de sucesso; um erro documentado usa o corpo
daquele erro; senão, um corpo JSON genérico com o status. Flag repetível; cada
status deve estar em `100–599`.

> Combinar duas flags que sobrescrevem o status no mesmo endpoint (ex.: `--fail`
> + `--sequence`) faz a última da chain vencer.

---

## Requisitos

- [Go 1.26+](https://go.dev/dl/) (só necessário para compilar/instalar a partir do código).

---

## Instalação

### Opção A — instalar o binário (recomendado)

```bash
go install github.com/Claudio712005/mock-smith/cmd/mocksmith@latest
```

Isso coloca o binário `mocksmith` no diretório bin do Go. Garanta que ele está
no `PATH`:

- **Linux / macOS** — adicione ao `~/.bashrc` / `~/.zshrc`:

  ```bash
  export PATH="$PATH:$(go env GOPATH)/bin"
  ```

- **Windows (PowerShell)** — o `go install` usa `%USERPROFILE%\go\bin`. Adicione
  uma vez:

  ```powershell
  setx PATH "$($env:PATH);$($env:USERPROFILE)\go\bin"
  ```

  Reinicie o terminal depois.

### Opção B — compilar a partir do código

```bash
git clone https://github.com/Claudio712005/mock-smith.git
cd mock-smith
go build -o mocksmith ./cmd/mocksmith
```

### Opção C — rodar sem instalar

A partir da raiz do projeto (pasta que contém o `go.mod`):

```bash
go run ./cmd/mocksmith run examples/petstore.yaml
```

---

## Uso

```bash
mocksmith run <spec> [flags]
```

`<spec>` é o caminho para um arquivo OpenAPI/Swagger (`.yaml`, `.yml` ou `.json`).

### Exemplos

```bash
# porta padrão :8080
mocksmith run examples/petstore.yaml

# YAML ou JSON funcionam igual
mocksmith run examples/petstore.json

# porta customizada
mocksmith run examples/petstore.yaml --addr :9090

# profile de runtime (happy, sad, resilience, chaos)
mocksmith run examples/petstore.yaml --profile chaos

# forçar 503 nos endpoints que o mapeiam; o resto segue o profile
mocksmith run examples/petstore.yaml --force-status 503
```

Chame de outro terminal:

```bash
curl http://localhost:8080/pets/1
curl -X POST http://localhost:8080/payments
```

---

## Endpoints de introspecção (admin)

Além das rotas geradas da spec, o MockSmith expõe rotas internas sob o prefixo
`/__mocksmith` para inspecionar o que foi carregado. Não fazem parte do seu
contrato OpenAPI.

### `GET /__mocksmith/endpoints`

Lista todos os endpoints carregados.

```bash
curl http://localhost:8080/__mocksmith/endpoints
```

```json
{
  "count": 6,
  "endpoints": [
    {
      "method": "POST",
      "path": "/payments",
      "operationId": "createPayment",
      "summary": "Create a payment",
      "statuses": [201, 503]
    }
  ]
}
```

| Campo               | Descrição                                            |
|---------------------|------------------------------------------------------|
| `count`             | Total de endpoints carregados.                       |
| `endpoints[].method`| Método HTTP.                                          |
| `endpoints[].path`  | Path no formato OpenAPI (ex.: `/pets/{petId}`).      |
| `endpoints[].operationId` | `operationId` da spec, quando presente.        |
| `endpoints[].summary`     | Resumo da operação, quando presente.           |
| `endpoints[].statuses`    | Status documentados (sucesso + erros).         |

### `GET /__mocksmith/endpoint`

Detalha um endpoint específico, identificado por `method` + `path` via query.

| Query    | Obrigatório | Descrição                                         |
|----------|-------------|---------------------------------------------------|
| `method` | sim         | Método HTTP (case-insensitive).                   |
| `path`   | sim         | Path exato no formato OpenAPI (ex.: `/pets/{petId}`). |

```bash
curl "http://localhost:8080/__mocksmith/endpoint?method=POST&path=/payments"
```

```json
{
  "method": "POST",
  "path": "/payments",
  "operationId": "createPayment",
  "summary": "Create a payment",
  "success": {
    "status": 201,
    "contentType": "application/json",
    "hasBody": true,
    "hasExample": false
  },
  "errors": [
    { "status": 503, "contentType": "application/json", "hasBody": true, "hasExample": false }
  ]
}
```

Respostas:

- `200` — endpoint encontrado.
- `400` — falta `method` ou `path`.
- `404` — nenhum endpoint casa com `method` + `path`.

---

## Admin API — overrides em runtime

Mude o comportamento de um endpoint **sem reiniciar** o servidor, via rotas sob
`/__mocksmith/runtime`. Um override sobrepõe o profile e as flags estáticas e
vale até ser removido.

### `POST /__mocksmith/runtime` — define um override

```json
{ "endpoint": "/payments", "status": 503, "latencyMs": 2000, "rate": 0.2 }
```

| Campo       | Obrigatório | Descrição                                                     |
|-------------|-------------|---------------------------------------------------------------|
| `endpoint`  | sim         | Path do template OpenAPI (ex.: `/pets/{petId}`). Deve existir. |
| `status`    | —           | Status HTTP a forçar (`100–599`). Igual ao sucesso documentado usa o corpo de sucesso. |
| `latencyMs` | —           | Atraso em milissegundos antes de responder.                   |
| `rate`      | —           | Fração `0–1` das requisições afetadas pelo `status` (vazio/`0` = sempre). |

Pelo menos um de `status` ou `latencyMs`. `POST` **substitui** o override inteiro
do endpoint — combine status e latência numa só chamada. Respostas: `200` (ok),
`400` (corpo/valores inválidos), `404` (endpoint inexistente).

### `GET /__mocksmith/runtime` — lista os overrides ativos

### `DELETE /__mocksmith/runtime` — remove

- `?endpoint=/payments` remove um (`404` se não existir);
- sem query, limpa todos e devolve `{ "cleared": N }`.

### CLI `mocksmith inject`

Atalho que conversa com a Admin API de um servidor rodando (`--addr`, padrão
`:8080`).

```bash
mocksmith inject "/payments:503(20%)"        # 20% das req → 503
mocksmith inject "/payments:503" --latency-ms 2000
mocksmith inject /payments --latency-ms 2000 # só latência
mocksmith inject --list
mocksmith inject --remove /payments
mocksmith inject --clear
```

---

## Arquivo de configuração (`mocksmith.yaml`)

Em vez de uma linha de comando gigante, declare tudo num YAML e rode com
`--config`. Sem caminho, ele procura `mocksmith.yaml` no diretório atual; com
caminho, usa o arquivo indicado.

```bash
mocksmith run --config              # carrega ./mocksmith.yaml
mocksmith run --config dev.yaml     # arquivo específico
```

```yaml
spec: petstore.yaml          # caminho da spec (relativo ao cwd)
addr: ":8080"
profile: resilience          # happy | sad | resilience | chaos
forceStatus: 0               # 0 = off

# Comportamentos por endpoint (chave = template OpenAPI)
endpoints:
  /payments:
    fail: 503                # sempre 503
    slow: 800ms              # + 800ms de latência
  /pets:
    timeout: 20%             # 20% viram timeout (504)
    corrupt: 10%             # 10% recebem JSON malformado
  /pets/{petId}:
    sequence: [404, 404, 200]  # 1ª/2ª → 404, 3ª+ → 200

# Overrides iniciais da Admin API (estado de runtime no boot)
overrides:
  /users/{id}:
    status: 503
    latencyMs: 1000
    rate: 0.5                # opcional: fração afetada pelo status
```

Campos de comportamento por endpoint: `fail` (int), `slow`/`timeout`/`corrupt`
(strings, mesmas das flags), `sequence` (lista de int). O exemplo completo em
[examples/mocksmith.yaml](examples/mocksmith.yaml) cobre todas as features sobre a
spec rica [examples/shop-api.yaml](examples/shop-api.yaml) (26 endpoints, todos os
métodos HTTP, formatos, arrays, aninhamento, enums, composição). Rode direto:

```bash
cd examples && mocksmith run --config
```

**Precedência:** o config é a base; **as flags da CLI vencem**. Escalares
(`--addr`, `--profile`, `--force-status`) sobrescrevem; listas por endpoint
(`--slow`, `--fail`, …) **somam** ao que veio do config. A `spec` pode vir do arg
posicional ou do campo `spec:`. Chave desconhecida no YAML é erro.

```bash
# config define profile: resilience, mas a flag força happy nesta execução
mocksmith run --config --profile happy
```

---

## Comandos & flags

### `mocksmith run <spec>`

Sobe um servidor de mock a partir de uma spec OpenAPI.

| Flag           | Padrão    | Descrição                                       |
|----------------|-----------|-------------------------------------------------|
| `--addr`       | `:8080`   | Endereço em que o servidor escuta.              |
| `--profile`    | `happy`   | Profile de runtime: `happy`, `sad`, `resilience`, `chaos` (ver [Profiles](#profiles)). |
| `--force-status` | `0`     | Força esse status nos endpoints que o documentam; os demais seguem o profile (`0` = off). Ver [Forçar um status](#forçar-um-status---force-status). |
| `--slow`       | —         | `[path=]duração`, ex.: `/pets=2s`. Repetível. Ver [Flags por endpoint](#flags-por-endpoint). |
| `--fail`       | —         | `[path=]status`, ex.: `/payments=503`. Repetível. |
| `--timeout`    | —         | `[path=]taxa`, ex.: `/auth=20%`. Repetível. |
| `--corrupt`    | —         | `[path=]taxa`, ex.: `/users=10%`. Repetível. |
| `--sequence`   | —         | `[path=]s1,s2,...`, ex.: `/jobs=202,202,200`. Repetível. Ver [Sequências stateful](#sequências-stateful---sequence). |
| `--config`     | —         | Carrega um YAML de config; sem valor usa `mocksmith.yaml`. Ver [Arquivo de configuração](#arquivo-de-configuração-mocksmithyaml). |
| `-h`, `--help` | —         | Ajuda do comando.                               |

Com `--config`, o argumento `[spec]` é opcional (vem do campo `spec:`).

### `mocksmith inject [...]`

Muda o comportamento de um servidor em execução via [Admin API](#admin-api--overrides-em-runtime).

| Flag           | Padrão    | Descrição                                       |
|----------------|-----------|-------------------------------------------------|
| `--addr`       | `:8080`   | Endereço do servidor em execução.               |
| `--latency-ms` | `0`       | Latência em milissegundos a injetar.            |
| `--list`       | `false`   | Lista os overrides ativos.                      |
| `--remove`     | —         | Remove o override de um path.                   |
| `--clear`      | `false`   | Remove todos os overrides.                      |

### Global

| Flag           | Descrição                  |
|----------------|----------------------------|
| `-h`, `--help` | Ajuda de qualquer comando. |

```bash
mocksmith --help
mocksmith run --help
```

---

## Execução por SO

Os comandos são idênticos entre plataformas; só muda a configuração do PATH
(veja [Instalação](#instalação)).

### Linux / macOS

```bash
mocksmith run examples/petstore.yaml --addr :8080
```

### Windows (PowerShell / CMD)

```powershell
mocksmith.exe run examples\petstore.yaml --addr :8080
```

Para gerar um binário Windows a partir de outro SO:

```bash
GOOS=windows GOARCH=amd64 go build -o mocksmith.exe ./cmd/mocksmith
```

---

## Estrutura do projeto

```txt
cmd/mocksmith/        ponto de entrada da CLI
internal/
  cli/                comandos cobra (root, run, inject)
  app/                wiring do runtime (carga → descoberta → serve)
  config/             parse do mocksmith.yaml → Options
  openapi/            carga da spec + descoberta de endpoints
  scenario/           engine de profiles (sorteio de comportamento por peso)
  interceptor/        pipeline de comportamento (latência, falha, timeout, corrupção)
  transport/http/     servidor chi e handlers das requisições
  domain/             modelos Endpoint / ResponseSpec
  faker/              geração de dado falso a partir do schema
examples/             specs de exemplo (petstore, shop-api, springboot-legacy) + mocksmith.yaml
test/                 testes end-to-end (spec real → servidor HTTP vivo)
.github/workflows/    pipeline de CI (build, lint, testes)
```

---

## Dependências

Bibliotecas diretas (ver [`go.mod`](go.mod) para versões exatas):

| Lib | Versão | Papel no MockSmith |
|-----|--------|--------------------|
| [`getkin/kin-openapi`](https://github.com/getkin/kin-openapi) | `v0.139.0` | Parse, validação e resolução de `$ref` de documentos OpenAPI 3 (YAML/JSON). Modela schemas/respostas usados na descoberta e na geração de corpo. Usada em `internal/openapi` e `internal/faker`. |
| [`go-chi/chi/v5`](https://github.com/go-chi/chi) | `v5.3.0` | Roteador HTTP. Templates OpenAPI (`/users/{id}`) casam direto com a sintaxe do chi. Middlewares `RequestID`, `Logger`, `Recoverer`. Usada em `internal/transport/http`. |
| [`spf13/cobra`](https://github.com/spf13/cobra) | `v1.10.2` | Framework de CLI: comando raiz `mocksmith`, subcomando `run`, flags e help. Usada em `internal/cli`. |
| [`brianvoe/gofakeit/v7`](https://github.com/brianvoe/gofakeit) | `v7.15.0` | Geração de dado falso ciente de formato (email, uuid, uri, date/date-time, ipv4/ipv6, números, strings com bounds). Usada em `internal/faker`. |
| [`gopkg.in/yaml.v3`](https://gopkg.in/yaml.v3) | `v3.0.1` | Parse do arquivo de config `mocksmith.yaml`. Usada em `internal/config`. |

As demais entradas do `go.mod` são dependências **indiretas** (transitivas) puxadas pelas acima — não importadas direto pelo código.

---

## Desenvolvimento & testes

Suíte de testes unitários (`internal/...`) + end-to-end (`test/`, sobe um servidor
HTTP vivo a partir da spec de exemplo).

```bash
# todos os testes
go test ./...

# com race detector e cobertura
go test -race -cover ./...

# relatório de cobertura por função
go test -coverprofile=coverage.out ./...
go tool cover -func=coverage.out

# só os e2e
go test ./test/...
```

### CI

Toda push/PR para `main` dispara o workflow [`.github/workflows/ci.yml`](.github/workflows/ci.yml),
que roda no Go 1.26.3:

1. **gofmt** — falha se houver arquivo não formatado.
2. **`go vet ./...`** — análise estática.
3. **`go test -race -coverprofile`** — testes com race detector e cobertura.
4. **`go build`** — garante que o binário compila.

Antes de abrir PR, rode localmente o mesmo conjunto:

```bash
gofmt -l .          # deve sair vazio
go vet ./...
go test -race ./...
go build -o mocksmith ./cmd/mocksmith
```

---

## Licença

Veja [LICENSE](LICENSE).
