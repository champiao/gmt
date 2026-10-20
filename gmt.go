package gmt

import (
	"github.com/champiao/gmt/internal"
	"github.com/champiao/gmt/pkg/cli"
	"gorm.io/gorm"
)

// Interfaces para migrações e seeds
type Migration = internal.Migration
type Seed = internal.Seed
type Depender = internal.Depender
type Runner = internal.Runner

// Register registra uma migração no sistema GMT
func Register(m Migration) {
	internal.Register(m)
}

// RegisterSeed registra uma seed no sistema GMT
func RegisterSeed(s Seed) {
	internal.RegisterSeed(s)
}

// NewRunner cria um executor de migrações e seeds
func NewRunner(db *gorm.DB) *Runner {
	return internal.NewRunner(db)
}

// RunDirect executa diretamente os comandos quando disparado via runner
func RunDirect() {
	cli.RunDirect()
}

// Run executa a CLI do GMT
func Run(args []string) error {
	return cli.Run(args)
}
