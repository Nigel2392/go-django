package benchmarks_test

import (
	"fmt"
	"log"
	"os"
	"testing"

	"github.com/Nigel2392/go-django/djester/quest"
	"github.com/Nigel2392/go-django/djester/testdb"
	django "github.com/Nigel2392/go-django/src"
	"github.com/Nigel2392/go-django/src/core/attrs"
	"github.com/Nigel2392/go-django/src/core/logger"
)

const (
	COUNT            = 500
	BOOKS_PER_AUTHOR = 1
	TOTAL_BOOKS      = (COUNT * BOOKS_PER_AUTHOR)
	TOTAL_COUNT      = COUNT + TOTAL_BOOKS

	M2M_SOURCES_COUNT      = 100
	M2M_TARGETS_COUNT      = 100
	M2M_TARGETS_PER_SOURCE = 10
	TOTAL_M2M_THROUGHS     = M2M_SOURCES_COUNT * M2M_TARGETS_PER_SOURCE
)

var (
	fldCnfPrimary = &attrs.FieldConfig{
		Primary: true,
	}
)

type fatalLogger struct {
}

func (l *fatalLogger) Helper() {}

func (l *fatalLogger) Fatal(args ...any) {
	log.Fatal(args...)
}

func (l *fatalLogger) Logf(s string, args ...any) {
	log.Printf(s, args...)
}

func (l *fatalLogger) Fatalf(s string, args ...any) {
	log.Fatalf(s, args...)
}

func TestMain(m *testing.M) {
	attrs.ALLOW_METHOD_CHECKS = false

	log.Print("initialising database...")

	var _, db = testdb.Open()
	var settings = map[string]interface{}{
		django.APPVAR_DATABASE: db,
	}

	logger.Setup(&logger.Logger{
		Level:       logger.DBG,
		WrapPrefix:  logger.ColoredLogWrapper,
		OutputDebug: os.Stdout,
		OutputInfo:  os.Stdout,
		OutputWarn:  os.Stdout,
		OutputError: os.Stdout,
	})

	django.App(django.Configure(settings))

	var tables = quest.Table(&fatalLogger{},
		&BenchmarkAuthor{},
		&BenchmarkBook{},

		&BenchmarkM2MTarget{},
		&BenchmarkM2MSource{},
		&BenchmarkM2MThrough{},

		&BenchmarkO2OTarget{},
		&BenchmarkO2OMain{},
		&BenchmarkO2OThrough{},

		&BenchmarkO2ONoThroughTarget{},
		&BenchmarkO2ONoThroughMain{},
	)

	attrs.RegisterModel(&BenchmarkAuthorModel{})
	attrs.RegisterModel(&BenchmarkBookModel{})
	attrs.ResetDefinitions.Send(nil)

	tables.Create()
	defer tables.Drop()

	/*
		BOOK HAS FK TO AUTHORS
	*/

	log.Println("pre-initialising model objects to create")

	var (
		authors = make([]*BenchmarkAuthor, COUNT)
		books   = make([]*BenchmarkBook, TOTAL_BOOKS)
	)

	for i := range COUNT {
		authors[i] = &BenchmarkAuthor{
			Name: fmt.Sprintf("BenchmarkAuthor #%d", i),
		}
	}

	for i := range TOTAL_BOOKS {
		books[i] = &BenchmarkBook{
			Title:  fmt.Sprintf("BenchmarkBook #%d", i),
			Author: authors[i%COUNT],
		}
	}

	log.Println("pre-initialising M2M objects to create")

	var (
		m2mSources  = make([]*BenchmarkM2MSource, M2M_SOURCES_COUNT)
		m2mTargets  = make([]*BenchmarkM2MTarget, M2M_TARGETS_COUNT)
		m2mThroughs = make([]*BenchmarkM2MThrough, TOTAL_M2M_THROUGHS)
	)

	for i := range M2M_SOURCES_COUNT {
		m2mSources[i] = &BenchmarkM2MSource{
			Title: fmt.Sprintf("M2M Source #%d", i),
		}
	}

	for i := range M2M_TARGETS_COUNT {
		m2mTargets[i] = &BenchmarkM2MTarget{
			Name: fmt.Sprintf("M2M Target #%d", i),
		}
	}

	log.Printf("Creating %d M2M sources and %d M2M targets...", M2M_SOURCES_COUNT, M2M_TARGETS_COUNT)
	_, m2mSourceDelete := quest.CreateObjects(&fatalLogger{}, m2mSources)
	_, m2mTargetDelete := quest.CreateObjects(&fatalLogger{}, m2mTargets)

	var throughIdx = 0
	for i := range M2M_SOURCES_COUNT {
		for j := range M2M_TARGETS_PER_SOURCE {
			m2mThroughs[throughIdx] = &BenchmarkM2MThrough{
				SourceModel: m2mSources[i].ID,
				TargetModel: m2mTargets[(i+j)%M2M_TARGETS_COUNT].ID,
			}
			throughIdx++
		}
	}

	log.Printf("Creating %d authors...", len(authors))
	_, authorDelete := quest.CreateObjects(&fatalLogger{}, authors)

	// O2O setup
	var o2oMains = make([]*BenchmarkO2OMain, COUNT)
	var o2oTargets = make([]*BenchmarkO2OTarget, COUNT)
	var o2oThroughs = make([]*BenchmarkO2OThrough, COUNT)

	for i := range COUNT {
		o2oMains[i] = &BenchmarkO2OMain{
			Title: fmt.Sprintf("O2O Main #%d", i),
		}
		o2oTargets[i] = &BenchmarkO2OTarget{
			Name: fmt.Sprintf("O2O Target #%d", i),
		}
	}
	_, o2oMainDel := quest.CreateObjects(&fatalLogger{}, o2oMains)
	_, o2oTargetDel := quest.CreateObjects(&fatalLogger{}, o2oTargets)

	for i := range COUNT {
		o2oThroughs[i] = &BenchmarkO2OThrough{
			SourceModel: o2oMains[i].ID,
			TargetModel: o2oTargets[i].ID,
		}
	}
	_, o2oThroughDel := quest.CreateObjects(&fatalLogger{}, o2oThroughs)

	// O2O No Through setup
	var o2oNTMains = make([]*BenchmarkO2ONoThroughMain, COUNT)
	var o2oNTTargets = make([]*BenchmarkO2ONoThroughTarget, COUNT)

	for i := range COUNT {
		o2oNTTargets[i] = &BenchmarkO2ONoThroughTarget{
			Name: fmt.Sprintf("O2O NT Target #%d", i),
		}
	}
	_, o2oNTTargetDel := quest.CreateObjects(&fatalLogger{}, o2oNTTargets)

	for i := range COUNT {
		o2oNTMains[i] = &BenchmarkO2ONoThroughMain{
			Title:  fmt.Sprintf("O2O NT Main #%d", i),
			Target: o2oNTTargets[i],
		}
	}
	_, o2oNTMainDel := quest.CreateObjects(&fatalLogger{}, o2oNTMains)

	var bookDeletes = make([]func(int) error, 0, len(books)/1000)
	for i := 0; i < len(books); i += 1000 {
		end := i + 1000
		if end > len(books) {
			end = len(books)
		}

		log.Printf("Creating books, iteration %d / %d", end/1000, (len(books) / 1000))
		_, del := quest.CreateObjects(&fatalLogger{}, books[i:end])
		bookDeletes = append(bookDeletes, del)
	}

	var m2mThroughDeletes = make([]func(int) error, 0, len(m2mThroughs)/1000)
	for i := 0; i < len(m2mThroughs); i += 1000 {
		end := i + 1000
		if end > len(m2mThroughs) {
			end = len(m2mThroughs)
		}

		log.Printf("Creating M2M throughs, iteration %d / %d", end/1000, (len(m2mThroughs)/1000)+1)
		_, del := quest.CreateObjects(&fatalLogger{}, m2mThroughs[i:end])
		m2mThroughDeletes = append(m2mThroughDeletes, del)
	}

	log.Printf("Created %d M2M through records", TOTAL_M2M_THROUGHS)

	log.Printf(
		"Created %d authors and %d books (%d per author)", COUNT, TOTAL_BOOKS, BOOKS_PER_AUTHOR)

	code := m.Run()
	log.Println("Tearing down database...")
	for _, del := range bookDeletes {
		_ = del(0)
	}
	_ = authorDelete(0)

	// M2M Teardowns
	for _, del := range m2mThroughDeletes {
		_ = del(0)
	}
	_ = m2mTargetDelete(0)
	_ = m2mSourceDelete(0)

	// O2O Teardowns
	_ = o2oThroughDel(0)
	_ = o2oTargetDel(0)
	_ = o2oMainDel(0)

	// O2O NT Teardowns
	_ = o2oNTMainDel(0)
	_ = o2oNTTargetDel(0)

	os.Exit(code)
}
