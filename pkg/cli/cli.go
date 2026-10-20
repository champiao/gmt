package cli

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/champiao/gmt/config/db"
	"github.com/champiao/gmt/internal"
	"github.com/joho/godotenv"
)

// Run é o ponto de entrada principal chamado pela CLI gmt
func Run(args []string) error {
	if len(args) == 0 {
		printHelp()
		return nil
	}

	command := args[0]

	switch command {
	case "init":
		wd, err := os.Getwd()
		if err != nil {
			return err
		}
		if err := internal.InitProject(wd); err != nil {
			return err
		}
		fmt.Println("✔ GMT initialized successfully!")
		fmt.Println("  • Created folder: migration/versions")
		fmt.Println("  • Created folder: migration/seeds")
		return nil

	case "create":
		if len(args) < 2 {
			fmt.Println("usage: gmt create migration <name> | gmt create seed <name>")
			return nil
		}
		subCommand := args[1]
		switch subCommand {
		case "migration":
			if len(args) < 3 {
				fmt.Println("usage: gmt create migration <name>")
				return nil
			}
			wd, err := os.Getwd()
			if err != nil {
				return err
			}
			path, err := internal.CreateMigration(wd, args[2])
			if err != nil {
				return err
			}
			fmt.Printf("✔ Migration created: %s\n", path)
			return nil

		case "seed":
			if len(args) < 3 {
				fmt.Println("usage: gmt create seed <name>")
				return nil
			}
			wd, err := os.Getwd()
			if err != nil {
				return err
			}
			path, err := internal.CreateSeed(wd, args[2])
			if err != nil {
				return err
			}
			fmt.Printf("✔ Seed created: %s\n", path)
			return nil

		default:
			fmt.Printf("unknown target '%s'. Use 'migration' or 'seed'\n", subCommand)
			return nil
		}

	case "migration":
		// Suporte retrocompatível a: gmt migration create <name>
		if len(args) >= 2 && args[1] == "create" {
			if len(args) < 3 {
				fmt.Println("usage: gmt create migration <name>")
				return nil
			}
			wd, err := os.Getwd()
			if err != nil {
				return err
			}
			path, err := internal.CreateMigration(wd, args[2])
			if err != nil {
				return err
			}
			fmt.Printf("✔ Migration created: %s\n", path)
			return nil
		}
		fmt.Println("usage: gmt create migration <name>")
		return nil

	case "migrate", "seed", "rollback", "up", "down":
		// Se já estiver rodando dentro do runner gerado, executa diretamente
		if os.Getenv("GMT_DIRECT_RUN") == "true" {
			return executeDirect(args)
		}
		// Caso contrário, prepara o runner para compilar as migrações/seeds do projeto
		return dispatchRunner(args)

	case "help", "--help", "-h":
		printHelp()
		return nil

	default:
		// Se o usuário digitou 'seed create <name>'
		if command == "seed" && len(args) >= 2 && args[1] == "create" {
			if len(args) < 3 {
				fmt.Println("usage: gmt create seed <name>")
				return nil
			}
			wd, err := os.Getwd()
			if err != nil {
				return err
			}
			path, err := internal.CreateSeed(wd, args[2])
			if err != nil {
				return err
			}
			fmt.Printf("✔ Seed created: %s\n", path)
			return nil
		}

		fmt.Printf("Unknown command '%s'. Run 'gmt help' for usage.\n", command)
		return nil
	}
}

// RunDirect é chamado a partir do tmp_runner.go gerado
func RunDirect() {
	if err := executeDirect(os.Args[1:]); err != nil {
		fmt.Printf("❌ Erro: %v\n", err)
		os.Exit(1)
	}
}

func executeDirect(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("no command provided to runner")
	}

	wd, err := os.Getwd()
	if err != nil {
		return err
	}

	envPath := filepath.Join(wd, ".env")
	if _, err := os.Stat(envPath); err == nil {
		_ = godotenv.Load(envPath)
	}

	gormDB, err := db.Open()
	if err != nil {
		return fmt.Errorf("falha ao conectar ao banco de dados: %w", err)
	}

	runner := internal.NewRunner(gormDB)

	switch args[0] {
	case "migrate":
		return runner.Migrate()
	case "seed":
		return runner.Seed()
	case "rollback":
		return runner.Rollback()
	case "up":
		if len(args) < 2 {
			return fmt.Errorf("usage: gmt up <name>")
		}
		return runner.Up(args[1])
	case "down":
		if len(args) < 2 {
			return fmt.Errorf("usage: gmt down <name>")
		}
		return runner.Down(args[1])
	default:
		return fmt.Errorf("comando desconhecido no executor: %s", args[0])
	}
}

func dispatchRunner(args []string) error {
	wd, err := os.Getwd()
	if err != nil {
		return err
	}

	modPath, modName, err := findGoMod(wd)
	if err != nil {
		return fmt.Errorf("não foi possível localizar go.mod no projeto atual: %w\nExecute 'gmt' na raiz de um projeto Go.", err)
	}

	versionsDir := filepath.Join(wd, "migration", "versions")
	seedsDir := filepath.Join(wd, "migration", "seeds")

	hasVersions := hasGoFiles(versionsDir)
	hasSeeds := hasGoFiles(seedsDir)

	command := args[0]
	if (command == "migrate" || command == "rollback" || command == "up" || command == "down") && !hasVersions {
		fmt.Println("ℹ️  Nenhuma migration encontrada em migration/versions.")
		return nil
	}
	if command == "seed" && !hasSeeds {
		fmt.Println("ℹ️  Nenhuma seed encontrada em migration/seeds.")
		return nil
	}

	// Cria pasta temporária .gmt
	tmpDir := filepath.Join(wd, ".gmt")
	if err := os.MkdirAll(tmpDir, 0755); err != nil {
		return fmt.Errorf("falha ao criar pasta temporária .gmt: %w", err)
	}
	defer func() {
		_ = os.RemoveAll(tmpDir)
	}()

	var imports []string
	if hasVersions {
		rel, _ := filepath.Rel(filepath.Dir(modPath), versionsDir)
		pkgPath := filepath.ToSlash(filepath.Join(modName, rel))
		imports = append(imports, fmt.Sprintf(`	_ "%s"`, pkgPath))
	}
	if hasSeeds {
		rel, _ := filepath.Rel(filepath.Dir(modPath), seedsDir)
		pkgPath := filepath.ToSlash(filepath.Join(modName, rel))
		imports = append(imports, fmt.Sprintf(`	_ "%s"`, pkgPath))
	}

	runnerSource := fmt.Sprintf(`// Code generated by GMT. DO NOT EDIT.
package main

import (
%s
	"github.com/champiao/gmt"
)

func main() {
	gmt.RunDirect()
}
`, strings.Join(imports, "\n"))

	tmpRunnerFile := filepath.Join(tmpDir, "tmp_runner.go")
	if err := os.WriteFile(tmpRunnerFile, []byte(runnerSource), 0644); err != nil {
		return fmt.Errorf("falha ao gerar runner: %w", err)
	}

	cmdArgs := append([]string{"run", tmpRunnerFile}, args...)
	cmd := exec.Command("go", cmdArgs...)
	cmd.Dir = wd
	cmd.Env = append(os.Environ(), "GMT_DIRECT_RUN=true")
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			os.Exit(exitErr.ExitCode())
		}
		return err
	}

	return nil
}

func hasGoFiles(dir string) bool {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return false
	}
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".go") && !strings.HasSuffix(e.Name(), "_test.go") {
			return true
		}
	}
	return false
}

func findGoMod(startDir string) (string, string, error) {
	curr := startDir
	for {
		modFile := filepath.Join(curr, "go.mod")
		if _, err := os.Stat(modFile); err == nil {
			name, err := parseModuleName(modFile)
			if err != nil {
				return "", "", err
			}
			return modFile, name, nil
		}

		parent := filepath.Dir(curr)
		if parent == curr {
			break
		}
		curr = parent
	}
	return "", "", fmt.Errorf("go.mod não encontrado")
}

func parseModuleName(modFilePath string) (string, error) {
	file, err := os.Open(modFilePath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if strings.HasPrefix(line, "module ") {
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				return parts[1], nil
			}
		}
	}
	return "", fmt.Errorf("declaração de módulo não encontrada em %s", modFilePath)
}

func printHelp() {
	fmt.Println("GMT - Go Migration Tool")
	fmt.Println()
	fmt.Println("Uso:")
	fmt.Println("  gmt init                       Inicializa as pastas migration/versions e migration/seeds")
	fmt.Println("  gmt create migration <name>    Cria um novo arquivo de migration em migration/versions")
	fmt.Println("  gmt create seed <name>         Cria um novo arquivo de seed em migration/seeds")
	fmt.Println("  gmt migrate                    Executa todas as migrations pendentes (respeitando DependsOn)")
	fmt.Println("  gmt seed                       Executa todas as seeds pendentes em migration/seeds")
	fmt.Println("  gmt rollback                   Reverte as migrations aplicadas")
	fmt.Println("  gmt up <name>                  Aplica uma migration específica")
	fmt.Println("  gmt down <name>                Reverte uma migration específica")
	fmt.Println("  gmt help                       Exibe esta ajuda")
	fmt.Println()
	fmt.Println("Variáveis de Conexão (.env):")
	fmt.Println("  DATABASE_URL=postgres://user:pass@localhost:5432/dbname?sslmode=disable")
	fmt.Println("  Ou: DB_DRIVER=postgres DB_HOST=localhost DB_PORT=5432 DB_USER=... DB_PASSWORD=... DB_NAME=...")
	fmt.Println("  Drivers suportados: postgres, mysql, sqlite")
}
