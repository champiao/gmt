package internal

import (
	"errors"
	"fmt"
	"sort"
	"time"

	"github.com/champiao/gmt/models"
	"gorm.io/gorm"
)

type Migration interface {
	Name() string
	Up(db *gorm.DB) error
	Down(db *gorm.DB) error
}

type Seed interface {
	Name() string
	Seed(db *gorm.DB) error
}

type Depender interface {
	DependsOn() []string
}

type Runner struct {
	db         *gorm.DB
	migrations []Migration
	seeds      []Seed
}

func NewRunner(db *gorm.DB) *Runner {
	return &Runner{
		db:         db,
		migrations: All(),
		seeds:      AllSeeds(),
	}
}

func ensureTable(db *gorm.DB) error {
	if !db.Migrator().HasTable(&models.SchemaMigration{}) {
		return db.Migrator().CreateTable(&models.SchemaMigration{})
	}
	return nil
}

func ensureSeedTable(db *gorm.DB) error {
	if !db.Migrator().HasTable(&models.SchemaSeed{}) {
		return db.Migrator().CreateTable(&models.SchemaSeed{})
	}
	return nil
}

func (r *Runner) getExecuted() (map[string]bool, error) {
	var rows []models.SchemaMigration
	if err := r.db.Find(&rows).Error; err != nil {
		return nil, err
	}

	exec := make(map[string]bool, len(rows))
	for _, m := range rows {
		exec[m.Name] = true
	}
	return exec, nil
}

func (r *Runner) getExecutedSeeds() (map[string]bool, error) {
	var rows []models.SchemaSeed
	if err := r.db.Find(&rows).Error; err != nil {
		return nil, err
	}

	exec := make(map[string]bool, len(rows))
	for _, s := range rows {
		exec[s.Name] = true
	}
	return exec, nil
}

func (r *Runner) resolveOrder() ([]Migration, error) {
	graph := map[string][]string{}
	nodes := map[string]Migration{}

	for _, m := range r.migrations {
		name := m.Name()
		nodes[name] = m

		if d, ok := m.(Depender); ok {
			graph[name] = d.DependsOn()
		} else {
			graph[name] = nil
		}
	}

	for name, deps := range graph {
		for _, d := range deps {
			if _, exists := nodes[d]; !exists {
				return nil, fmt.Errorf("migration %s depends on unknown migration %s", name, d)
			}
		}
	}

	ordered := []Migration{}
	visited := map[string]bool{}
	tempMark := map[string]bool{}

	var visit func(string) error
	visit = func(n string) error {
		if tempMark[n] {
			return fmt.Errorf("cyclic dependency detected at migration %s", n)
		}
		if visited[n] {
			return nil
		}
		tempMark[n] = true

		for _, dep := range graph[n] {
			if dep == "" {
				continue
			}
			if err := visit(dep); err != nil {
				return err
			}
		}

		tempMark[n] = false
		visited[n] = true
		ordered = append(ordered, nodes[n])
		return nil
	}

	names := make([]string, 0, len(nodes))
	for name := range nodes {
		names = append(names, name)
	}
	sort.Strings(names)

	for _, name := range names {
		if !visited[name] {
			if err := visit(name); err != nil {
				return nil, err
			}
		}
	}

	return ordered, nil
}

func (r *Runner) Migrate() error {
	// Desabilita criação automática de constraints de FK durante migrations.
	r.db.Config.DisableForeignKeyConstraintWhenMigrating = true

	if err := ensureTable(r.db); err != nil {
		return err
	}

	executed, err := r.getExecuted()
	if err != nil {
		return err
	}

	ordered, err := r.resolveOrder()
	if err != nil {
		return err
	}

	if len(ordered) == 0 {
		fmt.Println("ℹ️  No migrations found to execute.")
		return nil
	}

	for _, m := range ordered {
		name := m.Name()

		if executed[name] {
			fmt.Printf("⏭  SKIP: %s (already executed)\n", name)
			continue
		}
		fmt.Printf("▶️  RUN:  %s\n", name)
		start := time.Now()

		tx := r.db.Begin()
		tx.Config.DisableForeignKeyConstraintWhenMigrating = true
		if err := m.Up(tx); err != nil {
			tx.Rollback()
			fmt.Printf("❌ ERROR in migration %s: %v\n", name, err)
			return err
		}

		if err := tx.Create(&models.SchemaMigration{
			Name:      name,
			AppliedAt: time.Now(),
		}).Error; err != nil {
			tx.Rollback()
			fmt.Printf("❌ ERROR saving migration record %s: %v\n", name, err)
			return err
		}

		if err := tx.Commit().Error; err != nil {
			fmt.Printf("❌ ERROR commit migration %s: %v\n", name, err)
			return err
		}

		fmt.Printf("✔ DONE: %s (%s)\n", name, time.Since(start))
	}

	fmt.Println("✔ All migrations executed successfully.")
	return nil
}

func (r *Runner) Seed() error {
	if err := ensureSeedTable(r.db); err != nil {
		return err
	}

	executed, err := r.getExecutedSeeds()
	if err != nil {
		return err
	}

	if len(r.seeds) == 0 {
		fmt.Println("ℹ️  No seeds found to execute.")
		return nil
	}

	// Ordena as seeds por nome cronológico
	ordered := make([]Seed, len(r.seeds))
	copy(ordered, r.seeds)
	sort.Slice(ordered, func(i, j int) bool {
		return ordered[i].Name() < ordered[j].Name()
	})

	for _, s := range ordered {
		name := s.Name()
		if executed[name] {
			fmt.Printf("⏭  SKIP: %s (already executed)\n", name)
			continue
		}

		fmt.Printf("▶️  SEED: %s\n", name)
		start := time.Now()

		tx := r.db.Begin()
		if err := s.Seed(tx); err != nil {
			tx.Rollback()
			fmt.Printf("❌ ERROR in seed %s: %v\n", name, err)
			return err
		}

		if err := tx.Create(&models.SchemaSeed{
			Name:      name,
			AppliedAt: time.Now(),
		}).Error; err != nil {
			tx.Rollback()
			fmt.Printf("❌ ERROR saving seed record %s: %v\n", name, err)
			return err
		}

		if err := tx.Commit().Error; err != nil {
			fmt.Printf("❌ ERROR commit seed %s: %v\n", name, err)
			return err
		}

		fmt.Printf("✔ DONE: %s (%s)\n", name, time.Since(start))
	}

	fmt.Println("✔ All seeds executed successfully.")
	return nil
}

func (r *Runner) Up(name string) error {
	for _, m := range r.migrations {
		if m.Name() == name {
			fmt.Printf("▶️  RUN UP: %s\n", name)
			if err := m.Up(r.db); err != nil {
				return err
			}
			if err := ensureTable(r.db); err != nil {
				return err
			}
			return r.db.Create(&models.SchemaMigration{
				Name:      name,
				AppliedAt: time.Now(),
			}).Error
		}
	}
	return errors.New("migration not found: " + name)
}

func (r *Runner) Down(name string) error {
	for _, m := range r.migrations {
		if m.Name() == name {
			fmt.Printf("▶️  RUN DOWN: %s\n", name)
			if err := m.Down(r.db); err != nil {
				return err
			}
			return r.db.Where("name = ?", name).Delete(&models.SchemaMigration{}).Error
		}
	}
	return errors.New("migration not found: " + name)
}

func (r *Runner) Rollback() error {
	sort.Slice(r.migrations, func(i, j int) bool {
		return r.migrations[i].Name() > r.migrations[j].Name()
	})

	for _, m := range r.migrations {
		if err := m.Down(r.db); err != nil {
			return err
		}
		r.db.Where("name = ?", m.Name()).Delete(&models.SchemaMigration{})
	}
	return nil
}
