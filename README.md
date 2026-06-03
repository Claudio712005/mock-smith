# MockSmith

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

> Status: **Fase 1 (MVP)**. Carrega a spec, descobre os endpoints e serve as
> respostas de sucesso documentadas. Engine de cenários, interceptors e as flags
> mais ricas estão no roadmap abaixo.

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
- **Profile happy-path** — retorna a resposta de sucesso documentada (menor 2xx),
  ou `204` quando nenhuma é documentada.

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

# profile explícito (MVP só suporta "happy")
mocksmith run examples/petstore.yaml --profile happy
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

## Comandos & flags

### `mocksmith run <spec>`

Sobe um servidor de mock a partir de uma spec OpenAPI.

| Flag           | Padrão    | Descrição                                       |
|----------------|-----------|-------------------------------------------------|
| `--addr`       | `:8080`   | Endereço em que o servidor escuta.              |
| `--profile`    | `happy`   | Profile de runtime. MVP só suporta `happy`.     |
| `-h`, `--help` | —         | Ajuda do comando.                               |

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
  cli/                comandos cobra (root, run)
  app/                wiring do runtime (carga → descoberta → serve)
  openapi/            carga da spec + descoberta de endpoints
  transport/http/     servidor chi e handlers das requisições
  domain/             modelos Endpoint / ResponseSpec
  faker/              geração de dado falso a partir do schema
examples/             specs de exemplo para teste
```

---

## Licença

Veja [LICENSE](LICENSE).
