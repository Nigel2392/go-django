package revisions_test

import (
	"context"
	"reflect"
	"strconv"
	"testing"
	"time"

	django "github.com/Nigel2392/go-django/src"
	"github.com/Nigel2392/go-django/src/contrib/revisions"
	"github.com/Nigel2392/go-django/src/core/attrs"
	"github.com/Nigel2392/go-django/src/core/contenttypes"
	"github.com/Nigel2392/go-django/src/djester/testdb"
	"github.com/pkg/errors"
)

type Artist struct {
	ID   int64 `attrs:"primary"`
	Name string
}

func (a *Artist) FieldDefs() attrs.Definitions {
	return attrs.AutoDefinitions(a, "ID", "Name")
}

type Laptop struct {
	ID         int64 `attrs:"primary"`
	Resolution string
}

func (l *Laptop) FieldDefs() attrs.Definitions {
	return attrs.AutoDefinitions(l, "ID", "Resolution")
}

type Bottle struct {
	ID     int64 `attrs:"primary"`
	Liters float64
}

func (b *Bottle) FieldDefs() attrs.Definitions {
	return attrs.AutoDefinitions(b, "ID", "Liters")
}

var (
	Artists = []Artist{
		{ID: 1, Name: "John"},
		{ID: 2, Name: "Doe"},
		{ID: 3, Name: "Jane"},
	}
	Laptops = []Laptop{
		{ID: 1, Resolution: "1080p"},
		{ID: 2, Resolution: "4k"},
		{ID: 3, Resolution: "720p"},
	}
	Bottles = []Bottle{
		{ID: 1, Liters: 1.5},
		{ID: 2, Liters: 1},
		{ID: 3, Liters: 0.75},
	}

	artistMap = make(map[int64]Artist)
	laptopMap = make(map[int64]Laptop)
	bottleMap = make(map[int64]Bottle)
)

func init() {
	for _, a := range Artists {
		artistMap[a.ID] = a
	}
	for _, l := range Laptops {
		laptopMap[l.ID] = l
	}
	for _, b := range Bottles {
		bottleMap[b.ID] = b
	}

	var contentObjects = []interface{}{&Artist{}, &Laptop{}, &Bottle{}}
	for _, obj := range contentObjects {
		var obj = obj
		contenttypes.Register(&contenttypes.ContentTypeDefinition{
			ContentObject: obj,
			GetLabel:      func(ctx context.Context) string { return reflect.TypeOf(obj).Name() },
			GetInstance: func(ctx context.Context, identifier interface{}) (interface{}, error) {
				var id int64
				switch v := identifier.(type) {
				case int:
					id = int64(v)
				case int64:
					id = v
				case string:
					var err error
					id, err = strconv.ParseInt(v, 10, 64)
					if err != nil {
						return nil, err
					}
				default:
					return nil, errors.New("invalid type")
				}

				switch obj.(type) {
				case *Artist:
					if artist, ok := artistMap[id]; ok {
						return &artist, nil
					}
				case *Laptop:
					if laptop, ok := laptopMap[id]; ok {
						return &laptop, nil
					}
				case *Bottle:
					if bottle, ok := bottleMap[id]; ok {
						return &bottle, nil
					}
				}

				return nil, errors.New("not found")
			},
		})
	}

	// var db, err = sql.Open("mysql", "root:my-secret-pw@tcp(127.0.0.1:3306)/django-pages-test?parseTime=true&multiStatements=true")
	var _, db = testdb.Open()
	var settings = django.Config(map[string]interface{}{
		django.APPVAR_DATABASE: db,
	})

	var app = django.App(
		django.AppSettings(settings),
		django.Apps(revisions.NewAppConfig),
		django.Flag(django.FlagSkipCmds, django.FlagSkipChecks),
	)

	if err := app.Initialize(); err != nil {
		panic(errors.Wrap(
			err, "failed to initialize app",
		))
	}
}

var (
	revIDCounter int64
)

func TestCreateRevision(t *testing.T) {
	var (
		artist = Artists[0]
		laptop = Laptops[0]
		bottle = Bottles[0]
	)

	time.Sleep(1 * time.Second)
	revIDCounter++
	artistRev, err := revisions.CreateRevision(&artist)
	if err != nil {
		t.Fatal(err)
	}

	time.Sleep(1 * time.Second)
	revIDCounter++
	laptopRev, err := revisions.CreateRevision(&laptop)
	if err != nil {
		t.Fatal(err)
	}

	time.Sleep(1 * time.Second)
	revIDCounter++
	bottleRev, err := revisions.CreateRevision(&bottle)
	if err != nil {
		t.Fatal(err)
	}

	t.Logf("Created revisions: %d, %d, %d", artistRev.ID, laptopRev.ID, bottleRev.ID)

	t.Run("TestLatestRevision", func(t *testing.T) {
		t.Run("Artist", func(t *testing.T) {
			t.Logf("Retrieving latest revision for artist %d", artist.ID)
			artistRev, err = revisions.LatestRevision(&artist)
			if err != nil {
				t.Fatal(err)
			}

			artistRevTest, err := artistRev.AsObject()
			if err != nil {
				t.Fatal(err)
			}

			if artistRevTest.(*Artist).ID != artist.ID {
				t.Fatalf("Expected artist ID %d, got %d", artist.ID, artistRevTest.(*Artist).ID)
			}

			if artistRevTest.(*Artist).Name != artist.Name {
				t.Fatalf("Expected artist name %s, got %s", artist.Name, artistRevTest.(*Artist).Name)
			}
		})

		t.Run("Laptop", func(t *testing.T) {
			t.Logf("Retrieving latest revision for laptop %d", laptop.ID)
			laptopRev, err = revisions.LatestRevision(&laptop)
			if err != nil {
				t.Fatal(err)
			}

			laptopRevTest, err := laptopRev.AsObject()
			if err != nil {
				t.Fatal(err)
			}

			if laptopRevTest.(*Laptop).ID != laptop.ID {
				t.Fatalf("Expected laptop ID %d, got %d", laptop.ID, laptopRevTest.(*Laptop).ID)
			}

			if laptopRevTest.(*Laptop).Resolution != laptop.Resolution {
				t.Fatalf("Expected laptop resolution %s, got %s", laptop.Resolution, laptopRevTest.(*Laptop).Resolution)
			}
		})

		t.Run("Bottle", func(t *testing.T) {
			t.Logf("Retrieving latest revision for bottle %d", bottle.ID)
			bottleRev, err = revisions.LatestRevision(&bottle)
			if err != nil {
				t.Fatal(err)
			}

			bottleRevTest, err := bottleRev.AsObject()
			if err != nil {
				t.Fatal(err)
			}

			if bottleRevTest.(*Bottle).ID != bottle.ID {
				t.Fatalf("Expected bottle ID %d, got %d", bottle.ID, bottleRevTest.(*Bottle).ID)
			}

			if bottleRevTest.(*Bottle).Liters != bottle.Liters {
				t.Fatalf("Expected bottle liters %f, got %f", bottle.Liters, bottleRevTest.(*Bottle).Liters)
			}
		})
	})

	t.Run("TestNewLatestRevision", func(t *testing.T) {

		t.Run("Artist", func(t *testing.T) {
			time.Sleep(1 * time.Second)
			t.Logf("Creating new revision for artist %d", artist.ID)
			artist.Name = "John Doe"
			revIDCounter++
			artistRevLatest, err := revisions.CreateRevision(&artist)
			if err != nil {
				t.Fatal(err)
			}

			t.Logf("Retrieving latest revision for artist %d", artist.ID)
			artistRev, err = revisions.LatestRevision(&artist)
			if err != nil {
				t.Fatal(err)
			}

			t.Logf("Latest revisions: %d", artistRev.ID)
			if artistRev.ID != revIDCounter || artistRevLatest.ID != artistRev.ID {
				t.Fatalf("Expected revision ID %d, got %d", revIDCounter, artistRev.ID)
			}

			artistRevTest, err := artistRev.AsObject()
			if err != nil {
				t.Fatal(err)
			}

			if artistRevTest.(*Artist).ID != artist.ID {
				t.Fatalf("Expected artist ID %d, got %d", artist.ID, artistRevTest.(*Artist).ID)
			}

			if artistRevTest.(*Artist).Name != artist.Name {
				t.Fatalf("Expected artist name %s, got %s", artist.Name, artistRevTest.(*Artist).Name)
			}
		})

		t.Run("Laptop", func(t *testing.T) {
			time.Sleep(1 * time.Second)
			t.Logf("Creating new revision for laptop %d", laptop.ID)
			laptop.Resolution = "720p"
			revIDCounter++
			laptopRevLatest, err := revisions.CreateRevision(&laptop)
			if err != nil {
				t.Fatal(err)
			}

			t.Logf("Retrieving latest revision for laptop %d", laptop.ID)
			laptopRev, err = revisions.LatestRevision(&laptop)
			if err != nil {
				t.Fatal(err)
			}

			t.Logf("Latest revisions: %d", laptopRev.ID)
			if laptopRev.ID != revIDCounter || laptopRevLatest.ID != laptopRev.ID {
				t.Fatalf("Expected revision ID %d, got %d", revIDCounter, laptopRev.ID)
			}

			laptopRevTest, err := laptopRev.AsObject()
			if err != nil {
				t.Fatal(err)
			}

			if laptopRevTest.(*Laptop).ID != laptop.ID {
				t.Fatalf("Expected laptop ID %d, got %d", laptop.ID, laptopRevTest.(*Laptop).ID)
			}

			if laptopRevTest.(*Laptop).Resolution != laptop.Resolution {
				t.Fatalf("Expected laptop resolution %s, got %s", laptop.Resolution, laptopRevTest.(*Laptop).Resolution)
			}
		})

		t.Run("Bottle", func(t *testing.T) {
			time.Sleep(1 * time.Second)
			t.Logf("Creating new revision for bottle %d", bottle.ID)
			bottle.Liters = 2
			revIDCounter++
			bottleRevLatest, err := revisions.CreateRevision(&bottle)
			if err != nil {
				t.Fatal(err)
			}

			t.Logf("Retrieving latest revision for bottle %d", bottle.ID)
			bottleRev, err = revisions.LatestRevision(&bottle)
			if err != nil {
				t.Fatal(err)
			}

			t.Logf("Latest revisions: %d", bottleRev.ID)
			if bottleRev.ID != revIDCounter || bottleRevLatest.ID != bottleRev.ID {
				t.Fatalf("Expected revision ID %d, got %d", revIDCounter, bottleRev.ID)
			}

			bottleRevTest, err := bottleRev.AsObject()
			if err != nil {
				t.Fatal(err)
			}

			if bottleRevTest.(*Bottle).ID != bottle.ID {
				t.Fatalf("Expected bottle ID %d, got %d", bottle.ID, bottleRevTest.(*Bottle).ID)
			}

			if bottleRevTest.(*Bottle).Liters != bottle.Liters {
				t.Fatalf("Expected bottle liters %f, got %f", bottle.Liters, bottleRevTest.(*Bottle).Liters)
			}
		})
	})

	t.Run("TestGetRevisions", func(t *testing.T) {
		t.Run("Artist", func(t *testing.T) {
			t.Logf("Retrieving all revisions for artist %d", artist.ID)
			artistRevs, err := revisions.GetRevisionsByObject(&artist, 1000, 0)
			if err != nil {
				t.Fatal(err)
			}

			if len(artistRevs) != 2 {
				t.Fatalf("Expected 2 revisions, got %d", len(artistRevs))
			}

			for i, rev := range artistRevs {
				t.Logf("Revision %d: %d", i+1, rev.ID)
			}

			if artistRevs[0].ID <= artistRevs[1].ID {
				t.Fatalf("Expected revision IDs to be in descending order (created at desc)")
			}
		})

		t.Run("Laptop", func(t *testing.T) {
			t.Logf("Retrieving all revisions for laptop %d", laptop.ID)
			laptopRevs, err := revisions.GetRevisionsByObject(&laptop, 1000, 0)
			if err != nil {
				t.Fatal(err)
			}

			if len(laptopRevs) != 2 {
				t.Fatalf("Expected 2 revisions, got %d", len(laptopRevs))
			}

			for i, rev := range laptopRevs {
				t.Logf("Revision %d: %d", i+1, rev.ID)
			}

			if laptopRevs[0].ID <= laptopRevs[1].ID {
				t.Fatalf("Expected revision IDs to be in descending order (created at desc)")
			}
		})

		t.Run("Bottle", func(t *testing.T) {
			t.Logf("Retrieving all revisions for bottle %d", bottle.ID)
			bottleRevs, err := revisions.GetRevisionsByObject(&bottle, 1000, 0)
			if err != nil {
				t.Fatal(err)
			}

			if len(bottleRevs) != 2 {
				t.Fatalf("Expected 2 revisions, got %d", len(bottleRevs))
			}

			for i, rev := range bottleRevs {
				t.Logf("Revision %d: %d", i+1, rev.ID)
			}

			if bottleRevs[0].ID <= bottleRevs[1].ID {
				t.Fatalf("Expected revision IDs to be in descending order (created at desc)")
			}
		})
	})

	t.Run("ListRevisions", func(t *testing.T) {
		var revs, err = revisions.ListRevisions(1000, 0)
		if err != nil {
			t.Fatal(err)
		}

		if len(revs) != 6 {
			t.Fatalf("Expected 6 revisions, got %d", len(revs))
		}

		for i, rev := range revs {
			if i > 0 && revs[i-1].ID <= rev.ID {
				t.Fatalf("Expected revision IDs to be in descending order (created at desc) %d <= %d", revs[i-1].ID, rev.ID)
			}
			t.Logf("Revision %d: %d", i+1, rev.ID)
		}
	})
}
