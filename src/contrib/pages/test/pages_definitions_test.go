package pages_test

import (
	"context"
	"fmt"
	"reflect"
	"testing"

	"github.com/Nigel2392/go-django/src/contrib/pages"
	"github.com/Nigel2392/go-django/src/core/contenttypes"
	"github.com/Nigel2392/go-django/src/core/trans"
)

type (
	TestPageParent1 pages.PageNode
	TestPageParent2 pages.PageNode
	TestPageParent3 pages.PageNode
	TestPageChild1  pages.PageNode
	TestPageChild2  pages.PageNode
	TestPageChild3  pages.PageNode
)

var (
	definition1 *pages.PageDefinition
	definition2 *pages.PageDefinition
	definition3 *pages.PageDefinition
	definition4 *pages.PageDefinition
	definition5 *pages.PageDefinition
	definition6 *pages.PageDefinition
	pkgPath     = reflect.TypeOf(TestPageParent1{}).PkgPath()
)

func init() {

	// Register definitions for each page type
	definition1 = &pages.PageDefinition{
		ContentTypeDefinition: &contenttypes.ContentTypeDefinition{
			ContentObject: &TestPageParent1{},
			GetLabel:      trans.S("TestPageParent1"),
		},
		DisallowCreate:  true,
		DisallowRoot:    true,
		ParentPageTypes: []string{},
		ChildPageTypes: []string{
			fmt.Sprintf("%s.TestPageChild1", pkgPath),
			fmt.Sprintf("%s.TestPageChild2", pkgPath),
		},
		GetForID: func(ctx context.Context, ref *pages.PageNode, id int64) (pages.Page, error) {
			return ref, nil
		},
	}
	definition2 = &pages.PageDefinition{
		ContentTypeDefinition: &contenttypes.ContentTypeDefinition{
			ContentObject: &TestPageParent2{},
			GetLabel:      trans.S("TestPageParent2"),
		},
		GetForID: func(ctx context.Context, ref *pages.PageNode, id int64) (pages.Page, error) {
			return ref, nil
		},
		ChildPageTypes: []string{
			fmt.Sprintf("%s.TestPageChild1", pkgPath),
			fmt.Sprintf("%s.TestPageChild2", pkgPath),
		},
	}
	definition3 = &pages.PageDefinition{
		ContentTypeDefinition: &contenttypes.ContentTypeDefinition{
			ContentObject: &TestPageParent3{},
			GetLabel:      trans.S("TestPageParent3"),
		},
		GetForID: func(ctx context.Context, ref *pages.PageNode, id int64) (pages.Page, error) {
			return ref, nil
		},
		ChildPageTypes: []string{
			fmt.Sprintf("%s.TestPageChild3", pkgPath),
		},
	}
	definition4 = &pages.PageDefinition{
		ContentTypeDefinition: &contenttypes.ContentTypeDefinition{
			ContentObject: &TestPageChild1{},
			GetLabel:      trans.S("TestPageChild1"),
		},
		GetForID: func(ctx context.Context, ref *pages.PageNode, id int64) (pages.Page, error) {
			return ref, nil
		},
		DisallowRoot: true,
		ParentPageTypes: []string{
			fmt.Sprintf("%s.TestPageParent1", pkgPath),
		},
	}
	definition5 = &pages.PageDefinition{
		ContentTypeDefinition: &contenttypes.ContentTypeDefinition{
			ContentObject: &TestPageChild2{},
			GetLabel:      trans.S("TestPageChild2"),
		},
		GetForID: func(ctx context.Context, ref *pages.PageNode, id int64) (pages.Page, error) {
			return ref, nil
		},
		DisallowRoot: true,
		ParentPageTypes: []string{
			fmt.Sprintf("%s.TestPageParent1", pkgPath),
			fmt.Sprintf("%s.TestPageParent2", pkgPath),
			fmt.Sprintf("%s.TestPageParent3", pkgPath),
		},
	}
	definition6 = &pages.PageDefinition{
		ContentTypeDefinition: &contenttypes.ContentTypeDefinition{
			ContentObject: &TestPageChild3{},
			GetLabel:      trans.S("TestPageChild3"),
		},
		GetForID: func(ctx context.Context, ref *pages.PageNode, id int64) (pages.Page, error) {
			return ref, nil
		},
		DisallowRoot: true,
		ParentPageTypes: []string{
			fmt.Sprintf("%s.TestPageParent1", pkgPath),
			fmt.Sprintf("%s.TestPageParent3", pkgPath),
		},
	}

	pages.Register(definition1)
	pages.Register(definition2)
	pages.Register(definition3)
	pages.Register(definition4)
	pages.Register(definition5)
	pages.Register(definition6)
}

func TestPageDefinitionsRegistered(t *testing.T) {
	var definitions = pages.ListDefinitions()
	if len(definitions) != 6 {
		t.Fatalf("expected 6 definitions, got %d", len(definitions))
	}

	var definitionMap = map[string]*pages.PageDefinition{
		fmt.Sprintf("%s.TestPageParent1", pkgPath): definition1,
		fmt.Sprintf("%s.TestPageParent2", pkgPath): definition2,
		fmt.Sprintf("%s.TestPageParent3", pkgPath): definition3,
		fmt.Sprintf("%s.TestPageChild1", pkgPath):  definition4,
		fmt.Sprintf("%s.TestPageChild2", pkgPath):  definition5,
		fmt.Sprintf("%s.TestPageChild3", pkgPath):  definition6,
	}

	var had = make(map[string]struct{})
	for _, def := range definitions {
		if _, exists := had[def.ContentType().TypeName()]; exists {
			t.Fatalf("definition %s already found", def.ContentType().TypeName())
		}
		had[def.ContentType().TypeName()] = struct{}{}
		if _, exists := definitionMap[def.ContentType().TypeName()]; !exists {
			t.Fatalf("definition %s not found in definitionMap", def.ContentType().TypeName())
		}
	}
}

func TestParent1SubpageTypes(t *testing.T) {
	var subPageTypes = map[string]*pages.PageDefinition{
		fmt.Sprintf("%s.TestPageChild1", pkgPath): definition4,
		fmt.Sprintf("%s.TestPageChild2", pkgPath): definition5,
	}

	var definition = pages.DefinitionForType(fmt.Sprintf("%s.TestPageParent1", pkgPath))
	if definition == nil {
		t.Fatal("definition not found")
	}

	var subPageTypesList = pages.ListDefinitionsForType(
		fmt.Sprintf("%s.TestPageParent1", pkgPath),
	)
	if len(subPageTypesList) != len(subPageTypes) {
		t.Fatalf("expected %d subPageTypes, got %d", len(subPageTypes), len(subPageTypesList))
	}

	for _, subPageType := range subPageTypesList {
		if _, exists := subPageTypes[subPageType.ContentType().TypeName()]; !exists {
			t.Fatalf("subPageType %s not found", subPageType.ContentType().TypeName())
		}
	}
}

func TestParent2SubpageTypes(t *testing.T) {
	var subPageTypes = map[string]*pages.PageDefinition{
		fmt.Sprintf("%s.TestPageChild1", pkgPath): definition4,
		fmt.Sprintf("%s.TestPageChild2", pkgPath): definition5,
	}

	var definition = pages.DefinitionForType(fmt.Sprintf("%s.TestPageParent2", pkgPath))
	if definition == nil {
		t.Fatal("definition not found")
	}

	var subPageTypesList = pages.ListDefinitionsForType(
		fmt.Sprintf("%s.TestPageParent2", pkgPath),
	)
	if len(subPageTypesList) != len(subPageTypes) {
		t.Fatalf("expected %d subPageTypes, got %d", len(subPageTypes), len(subPageTypesList))
	}

	for _, subPageType := range subPageTypesList {
		if _, exists := subPageTypes[subPageType.ContentType().TypeName()]; !exists {
			t.Fatalf("subPageType %s not found", subPageType.ContentType().TypeName())
		}
	}
}

func TestParent3SubpageTypes(t *testing.T) {
	var subPageTypes = map[string]*pages.PageDefinition{
		fmt.Sprintf("%s.TestPageChild3", pkgPath): definition6,
	}

	var definition = pages.DefinitionForType(fmt.Sprintf("%s.TestPageParent3", pkgPath))
	if definition == nil {
		t.Fatal("definition not found")
	}

	var subPageTypesList = pages.ListDefinitionsForType(
		fmt.Sprintf("%s.TestPageParent3", pkgPath),
	)
	if len(subPageTypesList) != len(subPageTypes) {
		t.Fatalf("expected %d subPageTypes, got %d", len(subPageTypes), len(subPageTypesList))
	}

	for _, subPageType := range subPageTypesList {
		if _, exists := subPageTypes[subPageType.ContentType().TypeName()]; !exists {
			t.Fatalf("subPageType %s not found", subPageType.ContentType().TypeName())
		}
	}
}

func TestListRootDefinitions(t *testing.T) {
	var rootDefinitions = pages.ListRootDefinitions()

	var rootDefinitionTypes = map[string]*pages.PageDefinition{
		fmt.Sprintf("%s.TestPageParent2", pkgPath): definition2,
		fmt.Sprintf("%s.TestPageParent3", pkgPath): definition3,
	}

	if len(rootDefinitions) != len(rootDefinitionTypes) {
		t.Fatalf("expected %d rootDefinitions, got %d", len(rootDefinitionTypes), len(rootDefinitions))
	}

	for _, rootDefinition := range rootDefinitions {
		if _, exists := rootDefinitionTypes[rootDefinition.ContentType().TypeName()]; !exists {
			t.Fatalf("rootDefinition %s not found", rootDefinition.ContentType().TypeName())
		}
	}
}

func TestListCreatableDefinitions(t *testing.T) {
	var creatableDefinitions = pages.ListDefinitions()
	creatableDefinitions = pages.FilterCreatableDefinitions(
		creatableDefinitions,
	)
	if len(creatableDefinitions) != 5 {
		t.Fatalf("expected 3 creatableDefinitions, got %d", len(creatableDefinitions))
	}

	var creatableDefinitionTypes = map[string]*pages.PageDefinition{
		fmt.Sprintf("%s.TestPageParent2", pkgPath): definition2,
		fmt.Sprintf("%s.TestPageParent3", pkgPath): definition3,
		fmt.Sprintf("%s.TestPageChild1", pkgPath):  definition4,
		fmt.Sprintf("%s.TestPageChild2", pkgPath):  definition5,
		fmt.Sprintf("%s.TestPageChild3", pkgPath):  definition6,
	}

	for _, creatableDefinition := range creatableDefinitions {
		if _, exists := creatableDefinitionTypes[creatableDefinition.ContentType().TypeName()]; !exists {
			t.Fatalf("creatableDefinition %s not found", creatableDefinition.ContentType().TypeName())
		}
	}
}
