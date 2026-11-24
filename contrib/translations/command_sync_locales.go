package translations

import (
	"context"
	"flag"
	"fmt"
	"slices"
	"strings"

	queries "github.com/Nigel2392/go-django/queries/src"
	"github.com/Nigel2392/go-django/queries/src/drivers"
	django "github.com/Nigel2392/go-django/src"
	"github.com/Nigel2392/go-django/src/core/command"
	"golang.org/x/text/language"
)

var syncLocalesCommand = &command.Cmd[[]string]{
	ID:   "sync-locales",
	Desc: "Save the configured locales to the database",
	FlagFunc: func(_ command.Manager, additional *[]string, f *flag.FlagSet) error {
		*additional = nil
		f.Func("locales", "Language tag to save alongside the configured locales (repeatable)", func(value string) error {
			*additional = append(*additional, value)
			return nil
		})
		return nil
	},
	Execute: func(_ command.Manager, additional []string, args []string) error {
		if len(args) != 0 {
			return fmt.Errorf("sync-locales does not accept positional arguments: %v", args)
		}

		var settings django.Settings
		if django.Global != nil {
			settings = django.Global.Settings
		}

		configured, err := configuredLocaleNames(settings)
		if err != nil {
			return err
		}

		names := slices.Concat(configured, additional)
		if len(names) == 0 {
			return nil
		}

		tags := make([]string, 0, len(names))
		pending := make(map[string]*Locale, len(names))
		for _, name := range names {
			tag, err := language.Parse(strings.TrimSpace(name))
			if err != nil {
				return fmt.Errorf("invalid locale %q: %w", name, err)
			}
			key := tag.String()
			if _, exists := pending[key]; !exists {
				tags = append(tags, key)
				pending[key] = &Locale{LanguageTag: LanguageTag(tag)}
			}
		}

		if settings == nil {
			return fmt.Errorf("sync-locales requires a database in the %s setting", django.APPVAR_DATABASE)
		}

		database, _ := settings.Get(django.APPVAR_DATABASE)
		if db, ok := database.(drivers.Database); !ok || db == nil {
			return fmt.Errorf("sync-locales requires a database in the %s setting", django.APPVAR_DATABASE)
		}

		ctx, tx, err := queries.StartTransaction(context.Background())
		if err != nil {
			return fmt.Errorf("sync-locales could not start a transaction: %w", err)
		}

		defer tx.Rollback(ctx)

		locales := queries.GetQuerySetWithContext(ctx, &Locale{})
		existing, err := locales.Filter("LanguageTag__in", tags).All()
		if err != nil {
			return fmt.Errorf("sync-locales could not read existing locales: %w", err)
		}

		for _, row := range existing {
			delete(pending, row.Object.LanguageTag.String())
		}

		if len(pending) == 0 {
			return nil
		}

		missing := make([]*Locale, 0, len(pending))
		for _, tag := range tags {
			if locale, ok := pending[tag]; ok {
				missing = append(missing, locale)
			}
		}

		if _, err := locales.BulkCreate(missing); err != nil {
			return fmt.Errorf("sync-locales could not save locales: %w", err)
		}

		if err := tx.Commit(ctx); err != nil {
			return fmt.Errorf("sync-locales could not commit locales: %w", err)
		}
		return nil
	},
}
