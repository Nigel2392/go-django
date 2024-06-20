package permissions

import (
	"net/http"

	"github.com/Nigel2392/django/core/contenttypes"
)

type permissionTesterRegistry struct {
	testersByObj  map[string][]PermissionTester
	testersByPerm map[string][]PermissionTester
	objectHooks   []func(r *http.Request, perm string, obj interface{}) bool
	stringHooks   []func(r *http.Request, perm string) bool
}

func (p *permissionTesterRegistry) new(tests []PermissionTester) *tester {
	if tests == nil {
		tests = make([]PermissionTester, 0)
	}
	return &tester{
		tests:       tests,
		objectHooks: p.objectHooks,
		stringHooks: p.stringHooks,
	}
}

func (p *permissionTesterRegistry) registerForObj(obj interface{}, tester PermissionTester) {
	var cType = contenttypes.NewContentType(obj)
	var typeName = cType.TypeName()
	if p.testersByObj == nil {
		p.testersByObj = make(map[string][]PermissionTester)
	}

	if _, exists := p.testersByObj[typeName]; !exists {
		p.testersByObj[typeName] = make([]PermissionTester, 0)
	}

	p.testersByObj[typeName] = append(p.testersByObj[typeName], tester)
}

func (p *permissionTesterRegistry) registerForPerm(perm string, tester PermissionTester) {
	if p.testersByPerm == nil {
		p.testersByPerm = make(map[string][]PermissionTester)
	}

	if _, exists := p.testersByPerm[perm]; !exists {
		p.testersByPerm[perm] = make([]PermissionTester, 0)
	}

	p.testersByPerm[perm] = append(p.testersByPerm[perm], tester)
}

func (p *permissionTesterRegistry) getTesterForObj(obj interface{}) PermissionTester {
	var cType = contenttypes.NewContentType(obj)
	var typeName = cType.TypeName()
	return p.new(p.testersByObj[typeName])
}

func (p *permissionTesterRegistry) getTesterForPerm(perm string) PermissionTester {
	return p.new(p.testersByPerm[perm])
}

func (p *permissionTesterRegistry) registerObjectHook(hook func(r *http.Request, perm string, obj interface{}) bool) {
	if p.objectHooks == nil {
		p.objectHooks = make([]func(r *http.Request, perm string, obj interface{}) bool, 0)
	}
	p.objectHooks = append(p.objectHooks, hook)
}

func (p *permissionTesterRegistry) registerStringHook(hook func(r *http.Request, perm string) bool) {
	if p.stringHooks == nil {
		p.stringHooks = make([]func(r *http.Request, perm string) bool, 0)
	}
	p.stringHooks = append(p.stringHooks, hook)
}
