package permissions

import (
	"net/http"

	"github.com/Nigel2392/django/core/contenttypes"
)

type permissionTesterRegistry struct {
	testersByObj  map[string][]PermissionTester
	testersByPerm map[string][]PermissionTester
	hooks         []func(r *http.Request, obj interface{}, perm string) bool
}

func (p *permissionTesterRegistry) new(tests []PermissionTester) *tester {
	if tests == nil {
		tests = make([]PermissionTester, 0)
	}
	return &tester{
		tests: tests,
		hooks: p.hooks,
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

func (p *permissionTesterRegistry) registerHook(hook func(r *http.Request, obj interface{}, perm string) bool) {
	if p.hooks == nil {
		p.hooks = make([]func(r *http.Request, obj interface{}, perm string) bool, 0)
	}
	p.hooks = append(p.hooks, hook)
}
