package internal

import (
	"testing"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

type mockMigration struct {
	name    string
	upErr   error
	downErr error
	upRan   bool
	downRan bool
	deps    []string
}

func (m *mockMigration) Name() string { return m.name }
func (m *mockMigration) Up(db *gorm.DB) error {
	m.upRan = true
	return m.upErr
}
func (m *mockMigration) Down(db *gorm.DB) error {
	m.downRan = true
	return m.downErr
}
func (m *mockMigration) DependsOn() []string { return m.deps }

type mockSeed struct {
	name    string
	seedErr error
	seedRan bool
}

func (s *mockSeed) Name() string { return s.name }
func (s *mockSeed) Seed(db *gorm.DB) error {
	s.seedRan = true
	return s.seedErr
}

func openTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("falha ao abrir sqlite em memória: %v", err)
	}
	return db
}

func TestResolveOrder(t *testing.T) {
	m1 := &mockMigration{name: "01_create_roles"}
	m2 := &mockMigration{name: "02_create_users", deps: []string{"01_create_roles"}}
	m3 := &mockMigration{name: "03_create_posts", deps: []string{"02_create_users"}}

	runner := &Runner{
		migrations: []Migration{m3, m1, m2},
	}

	ordered, err := runner.resolveOrder()
	if err != nil {
		t.Fatalf("resolveOrder falhou: %v", err)
	}

	expectedOrder := []string{"01_create_roles", "02_create_users", "03_create_posts"}
	for i, m := range ordered {
		if m.Name() != expectedOrder[i] {
			t.Errorf("ordem incorreta na posição %d: esperado %s, obteve %s", i, expectedOrder[i], m.Name())
		}
	}
}

func TestMigrateAndSeedExecution(t *testing.T) {
	db := openTestDB(t)

	m1 := &mockMigration{name: "20261020_000001_initial"}
	m2 := &mockMigration{name: "20261020_000002_update", deps: []string{"20261020_000001_initial"}}

	s1 := &mockSeed{name: "20261020_000001_users"}
	s2 := &mockSeed{name: "20261020_000002_settings"}

	runner := &Runner{
		db:         db,
		migrations: []Migration{m2, m1},
		seeds:      []Seed{s2, s1},
	}

	// 1. Executa migrações
	if err := runner.Migrate(); err != nil {
		t.Fatalf("Migrate falhou: %v", err)
	}
	if !m1.upRan || !m2.upRan {
		t.Errorf("todas as migrações deveriam ter sido executadas")
	}

	// 2. Executa novamente (deve dar skip sem erro)
	m1.upRan = false
	m2.upRan = false
	if err := runner.Migrate(); err != nil {
		t.Fatalf("Migrate idempotente falhou: %v", err)
	}
	if m1.upRan || m2.upRan {
		t.Errorf("nenhuma migração deveria ter sido executada na segunda vez")
	}

	// 3. Executa seeds
	if err := runner.Seed(); err != nil {
		t.Fatalf("Seed falhou: %v", err)
	}
	if !s1.seedRan || !s2.seedRan {
		t.Errorf("todas as seeds deveriam ter sido executadas")
	}

	// 4. Executa seeds novamente (deve dar skip sem erro)
	s1.seedRan = false
	s2.seedRan = false
	if err := runner.Seed(); err != nil {
		t.Fatalf("Seed idempotente falhou: %v", err)
	}
	if s1.seedRan || s2.seedRan {
		t.Errorf("nenhuma seed deveria ter sido executada na segunda vez")
	}
}
