package conf

import "github.com/Nigel2392/go-django/pkg/yml"

type MigrationTarget struct {
	Setup       string `yaml:"setup"`
	Destination string `yaml:"dst"`
}

type DatabaseConfig struct {
	Engine string `yaml:"engine"`
	DSN    string `yaml:"dsn"`
}

type MigrationConfig struct {
	Database   *DatabaseConfig                         `yaml:"db"`
	SourceDirs []string                                `yaml:"src"` // TODO: MAKE MIGRATOR COMPATIBLE WITH MULTIPLE SOURCE DIRECTORIES
	Targets    yml.OrderedMap[string, MigrationTarget] `yaml:"apps"`
}
