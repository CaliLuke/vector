package postgres

import (
	"context"
	"sync"
)

// Template migrates one disposable seed and supplies isolated databases to tests.
// Close it after all callers and their database connections have finished.
type Template struct {
	once    sync.Once
	migrate SeedMigration
	pair    *Pair
	err     error
}

func NewTemplate(migrate SeedMigration) *Template {
	return &Template{migrate: migrate}
}

func (template *Template) CloneDatabase(ctx context.Context) (*Database, error) {
	template.once.Do(func() {
		template.pair, template.err = StartUncloned(ctx, template.migrate)
	})
	if template.err != nil {
		return nil, template.err
	}
	return template.pair.CloneDatabase(ctx)
}

func (template *Template) Close() error {
	return template.pair.Close()
}
