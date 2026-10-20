package internal

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestFilesInitAndCreate(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "gmt-test-*")
	if err != nil {
		t.Fatalf("falha ao criar temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Testa InitProject
	if err := InitProject(tmpDir); err != nil {
		t.Fatalf("InitProject falhou: %v", err)
	}

	versionsDir := filepath.Join(tmpDir, "migration", "versions")
	seedsDir := filepath.Join(tmpDir, "migration", "seeds")

	if _, err := os.Stat(versionsDir); os.IsNotExist(err) {
		t.Errorf("diretório versions não foi criado: %v", err)
	}
	if _, err := os.Stat(seedsDir); os.IsNotExist(err) {
		t.Errorf("diretório seeds não foi criado: %v", err)
	}

	// Testa CreateMigration
	migPath, err := CreateMigration(tmpDir, "create users table")
	if err != nil {
		t.Fatalf("CreateMigration falhou: %v", err)
	}
	if !strings.Contains(migPath, "create_users_table.go") {
		t.Errorf("nome de arquivo inesperado: %s", migPath)
	}

	migContent, err := os.ReadFile(migPath)
	if err != nil {
		t.Fatalf("falha ao ler migration criada: %v", err)
	}
	if !strings.Contains(string(migContent), "package versions") {
		t.Errorf("migration gerada não contém package versions")
	}
	if !strings.Contains(string(migContent), "gmt.Register(") {
		t.Errorf("migration gerada não contém chamada de registro")
	}

	// Testa CreateSeed
	seedPath, err := CreateSeed(tmpDir, "populate admin user")
	if err != nil {
		t.Fatalf("CreateSeed falhou: %v", err)
	}
	if !strings.Contains(seedPath, "populate_admin_user.go") {
		t.Errorf("nome de arquivo inesperado: %s", seedPath)
	}

	seedContent, err := os.ReadFile(seedPath)
	if err != nil {
		t.Fatalf("falha ao ler seed criada: %v", err)
	}
	if !strings.Contains(string(seedContent), "package seeds") {
		t.Errorf("seed gerada não contém package seeds")
	}
	if !strings.Contains(string(seedContent), "gmt.RegisterSeed(") {
		t.Errorf("seed gerada não contém chamada de registro de seed")
	}
	if !strings.Contains(string(seedContent), "INSTRUÇÕES PARA CONFIGURAR A SEED") {
		t.Errorf("seed gerada não contém instruções para popular o banco")
	}
}
