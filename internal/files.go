package internal

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

func getLocation() *time.Location {
	loc, err := time.LoadLocation("America/Sao_Paulo")
	if err != nil {
		return time.Local
	}
	return loc
}

// InitProject inicializa a estrutura de pastas migration/versions e migration/seeds
func InitProject(baseDir string) error {
	versionsDir := filepath.Join(baseDir, "migration", "versions")
	seedsDir := filepath.Join(baseDir, "migration", "seeds")

	if err := os.MkdirAll(versionsDir, 0755); err != nil {
		return fmt.Errorf("falha ao criar pasta %s: %w", versionsDir, err)
	}

	if err := os.MkdirAll(seedsDir, 0755); err != nil {
		return fmt.Errorf("falha ao criar pasta %s: %w", seedsDir, err)
	}

	// Cria .gitkeep se as pastas estiverem vazias
	ensureGitkeep(versionsDir)
	ensureGitkeep(seedsDir)

	// Cria .env.example se não existir .env nem .env.example
	ensureEnvExample(baseDir)

	// Adiciona .gmt ao .gitignore se existir
	ensureGitignore(baseDir)

	return nil
}

func ensureGitkeep(dir string) {
	gitkeep := filepath.Join(dir, ".gitkeep")
	if _, err := os.Stat(gitkeep); os.IsNotExist(err) {
		_ = os.WriteFile(gitkeep, []byte(""), 0644)
	}
}

func ensureEnvExample(baseDir string) {
	envFile := filepath.Join(baseDir, ".env")
	envExample := filepath.Join(baseDir, ".env.example")

	if _, err := os.Stat(envFile); os.IsNotExist(err) {
		if _, err := os.Stat(envExample); os.IsNotExist(err) {
			content := `# Database configuration for GMT (Go Migration Tool)
DB_DRIVER=postgres
DB_HOST=localhost
DB_PORT=5432
DB_USER=postgres
DB_PASSWORD=postgres
DB_NAME=my_database
DB_SSLMODE=disable

# Or using full connection URL:
# DATABASE_URL=postgres://postgres:postgres@localhost:5432/my_database?sslmode=disable
# Or for SQLite:
# DB_DRIVER=sqlite
# DB_NAME=app.db
`
			_ = os.WriteFile(envExample, []byte(content), 0644)
		}
	}
}

func ensureGitignore(baseDir string) {
	gitignorePath := filepath.Join(baseDir, ".gitignore")
	if data, err := os.ReadFile(gitignorePath); err == nil {
		content := string(data)
		if !strings.Contains(content, ".gmt") {
			newContent := strings.TrimRight(content, "\n") + "\n\n# GMT temporary files\n.gmt/\n"
			_ = os.WriteFile(gitignorePath, []byte(newContent), 0644)
		}
	}
}

// CreateMigration cria um novo arquivo de migration em migration/versions
func CreateMigration(baseDir, name string) (string, error) {
	versionsDir := filepath.Join(baseDir, "migration", "versions")
	if err := os.MkdirAll(versionsDir, 0755); err != nil {
		return "", fmt.Errorf("falha ao criar diretório %s: %w", versionsDir, err)
	}

	snake := strings.ToLower(strings.ReplaceAll(strings.TrimSpace(name), " ", "_"))
	if snake == "" {
		snake = "migration"
	}

	loc := getLocation()
	timestamp := time.Now().In(loc).Format("20060102_150405")
	fileName := fmt.Sprintf("%s_%s.go", timestamp, snake)
	typeName := "M" + strings.ReplaceAll(timestamp+"_"+snake, "_", "")
	migrationName := timestamp + "_" + snake

	template := fmt.Sprintf(`package versions

import (
	"github.com/champiao/gmt"
	"gorm.io/gorm"
)

type %s struct{}

func (m %s) Name() string {
	return "%s"
}

func (m %s) Up(db *gorm.DB) error {
	// TODO: Implemente a criação/alteração de tabelas e esquemas.
	// Exemplo:
	// type User struct {
	//     ID        uint   `+"`gorm:\"primaryKey\"`"+`
	//     Name      string `+"`gorm:\"size:255;not null\"`"+`
	//     Email     string `+"`gorm:\"size:255;unique;not null\"`"+`
	// }
	// return db.AutoMigrate(&User{})

	return nil
}

func (m %s) Down(db *gorm.DB) error {
	// TODO: Implemente o rollback da migration.
	// Exemplo:
	// return db.Migrator().DropTable("users")

	return nil
}

func (m %s) DependsOn() []string {
	// Caso esta migração dependa da execução de outra antes, liste os nomes aqui.
	// Exemplo: return []string{"20261020_120000_previous_migration"}
	return []string{}
}

func init() {
	gmt.Register(%s{})
}
`, typeName, typeName, migrationName, typeName, typeName, typeName, typeName)

	filePath := filepath.Join(versionsDir, fileName)
	if err := os.WriteFile(filePath, []byte(template), 0644); err != nil {
		return "", fmt.Errorf("falha ao escrever arquivo de migração: %w", err)
	}

	return filePath, nil
}

// CreateSeed cria um novo arquivo de seed em migration/seeds
func CreateSeed(baseDir, name string) (string, error) {
	seedsDir := filepath.Join(baseDir, "migration", "seeds")
	if err := os.MkdirAll(seedsDir, 0755); err != nil {
		return "", fmt.Errorf("falha ao criar diretório %s: %w", seedsDir, err)
	}

	snake := strings.ToLower(strings.ReplaceAll(strings.TrimSpace(name), " ", "_"))
	if snake == "" {
		snake = "seed"
	}

	loc := getLocation()
	timestamp := time.Now().In(loc).Format("20060102_150405")
	fileName := fmt.Sprintf("%s_%s.go", timestamp, snake)
	typeName := "S" + strings.ReplaceAll(timestamp+"_"+snake, "_", "")
	seedName := timestamp + "_" + snake

	template := fmt.Sprintf(`package seeds

import (
	"github.com/champiao/gmt"
	"gorm.io/gorm"
)

type %s struct{}

func (s %s) Name() string {
	return "%s"
}

func (s %s) Seed(db *gorm.DB) error {
	// =====================================================================
	// INSTRUÇÕES PARA CONFIGURAR A SEED COM SEUS MODELOS DE DADOS:
	// =====================================================================
	// 1. Importe os modelos do seu projeto ou declare structs locais.
	// 2. Use a instância 'db' (*gorm.DB) para inserir ou atualizar os dados.
	// 3. Dica: use FirstOrCreate ou Create para popular os dados com segurança.
	//
	// Exemplo:
	// type User struct {
	//     ID    uint   `+"`gorm:\"primaryKey\"`"+`
	//     Name  string
	//     Email string `+"`gorm:\"uniqueIndex\"`"+`
	//     Role  string
	// }
	//
	// users := []User{
	//     {Name: "Administrador", Email: "admin@example.com", Role: "admin"},
	//     {Name: "Usuário Padrão", Email: "user@example.com", Role: "user"},
	// }
	//
	// for _, u := range users {
	//     if err := db.Where(User{Email: u.Email}).FirstOrCreate(&u).Error; err != nil {
	//         return err
	//     }
	// }
	// =====================================================================

	return nil
}

func init() {
	gmt.RegisterSeed(%s{})
}
`, typeName, typeName, seedName, typeName, typeName)

	filePath := filepath.Join(seedsDir, fileName)
	if err := os.WriteFile(filePath, []byte(template), 0644); err != nil {
		return "", fmt.Errorf("falha ao escrever arquivo de seed: %w", err)
	}

	return filePath, nil
}
