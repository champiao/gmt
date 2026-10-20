# GMT (Go Migration Tool)

Ferramenta CLI para gerenciar migrações de esquemas de banco de dados e seeds em aplicações Go com [GORM](https://gorm.io).

## 🚀 Instalação

Para instalar globalmente em seu sistema via Go:

```bash
go install github.com/champiao/gmt/cmd/gmt@latest
```

> Certifique-se de que o diretório `$GOPATH/bin` (geralmente `~/go/bin`) esteja incluído no seu `PATH`.

---

## 🛠️ Como Usar no seu Projeto

### 1. Inicializar o projeto
Navegue até a raiz do seu projeto Go onde deseja gerenciar as migrações e execute:

```bash
gmt init
```

Isso irá criar a estrutura:
- `migration/versions/`: onde ficam os arquivos de migrações (`UP` / `DOWN`).
- `migration/seeds/`: onde ficam os arquivos de seeds para popular o banco.
- `.env.example`: modelo de variáveis de configuração do banco de dados.

### 2. Configurar a Conexão com o Banco de Dados

Crie um arquivo `.env` na raiz do seu projeto:

```env
# Exemplo com Postgres:
DB_DRIVER=postgres
DB_HOST=localhost
DB_PORT=5432
DB_USER=postgres
DB_PASSWORD=secret
DB_NAME=meu_banco
DB_SSLMODE=disable

# Ou via URL completa:
# DATABASE_URL=postgres://postgres:secret@localhost:5432/meu_banco?sslmode=disable

# Ou usando SQLite:
# DB_DRIVER=sqlite
# DB_NAME=app.db
```

### 3. Criar Migrações

Para criar uma nova migração:

```bash
gmt create migration create_users_table
```

Isso gera um arquivo dentro de `migration/versions/` com o timestamp e template com métodos `Up`, `Down` e `DependsOn`.

### 4. Executar Migrações

Para executar todas as migrações pendentes em ordem respeitando dependências (`DependsOn`):

```bash
gmt migrate
```

- Migrações já executadas recebem `SKIP`.
- O histórico é salvo automaticamente na tabela `schema_migrations`.

### 5. Criar Seeds

Para criar uma nova seed:

```bash
gmt create seed initial_users
```

Isso gera um arquivo dentro de `migration/seeds/` com o esqueleto da seed e instruções comentadas demonstrando como usar modelos de dados para popular a base.

### 6. Executar Seeds

Para executar as seeds pendentes:

```bash
gmt seed
```

- Executa as seeds cadastradas em ordem cronológica dentro de transações seguras.
- Seeds já executadas recebem `SKIP`.
- O histórico de execução de seeds é mantido na tabela `schema_seeds`.

---

## 📋 Resumo dos Comandos

| Comando | Descrição |
|---|---|
| `gmt init` | Inicializa as pastas `migration/versions` e `migration/seeds` no projeto |
| `gmt create migration <nome>` | Cria um novo arquivo de migration em `migration/versions` |
| `gmt create seed <nome>` | Cria um novo arquivo de seed em `migration/seeds` |
| `gmt migrate` | Executa todas as migrations pendentes (respeitando `DependsOn`) |
| `gmt seed` | Executa todas as seeds pendentes |
| `gmt rollback` | Reverte as migrations aplicadas |
| `gmt up <nome>` | Aplica uma migration específica |
| `gmt down <nome>` | Reverte uma migration específica |
| `gmt help` | Exibe manual de ajuda com os comandos |

---

## 🗄️ Bancos de Dados Suportados

- **PostgreSQL** (`postgres`)
- **MySQL** (`mysql`)
- **SQLite** (`sqlite` - pure Go, não requer CGO)
