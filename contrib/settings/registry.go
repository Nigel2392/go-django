package settings

import (
	"cmp"
	"context"
	"fmt"
	"html/template"
	"maps"
	"slices"
	"sync/atomic"
)

type SettingsGroup struct {
	Icon        func(context.Context) template.HTML
	Label       func(context.Context) string
	Description func(context.Context) string
}

const DEFAULT_GROUP = "Default"

var reg = new(settingsRegistry{
	groupsM: make(map[string]*registeredGroup),
})

type registeredSetting struct {
	ordering     int
	internalName string
	setting      Setting
}

type registeredGroup struct {
	SettingsGroup
	registry  *settingsRegistry
	name      string
	ordering  int
	settingsM map[string]Setting
	settings  []registeredSetting
}

type settingsRegistry struct {
	locked  atomic.Bool
	groupsM map[string]*registeredGroup
	groups  []*registeredGroup
}

func (s *settingsRegistry) lock() {
	if !s.locked.CompareAndSwap(false, true) {
		panic("settings registry is already locked")
	}

	slices.SortStableFunc(s.groups, func(a, b *registeredGroup) int {
		if ordering := cmp.Compare(a.ordering, b.ordering); ordering != 0 {
			return ordering
		}

		return cmp.Compare(a.name, b.name)
	})

	for _, group := range s.groups {
		slices.SortStableFunc(group.settings, func(a, b registeredSetting) int {
			if ordering := cmp.Compare(a.ordering, b.ordering); ordering != 0 {
				return ordering
			}

			return cmp.Compare(a.internalName, b.internalName)
		})
	}
}

func (s *settingsRegistry) registerGroup(ordering int, groupName string, group SettingsGroup) {
	if s.locked.Load() {
		panic("settings registry is locked, no settings may be registered anymore")
	}

	_, ok := s.groupsM[groupName]
	if ok {
		panic(fmt.Sprintf("setting %q was already registered in %v", groupName, maps.Keys(s.groupsM)))
	}

	_group := &registeredGroup{
		registry:      s,
		ordering:      ordering,
		name:          groupName,
		SettingsGroup: group,
		settingsM:     make(map[string]Setting),
		settings:      make([]registeredSetting, 0),
	}
	s.groupsM[groupName] = _group
	s.groups = append(s.groups, _group)
}

func (s *settingsRegistry) registerSetting(ordering int, groupName, name string, setting Setting) {
	if s.locked.Load() {
		panic("settings registry is locked, no settings may be registered anymore")
	}

	grp, ok := s.groupsM[groupName]
	if !ok {
		panic(fmt.Sprintf("group %q not fount in %v", groupName, maps.Keys(s.groupsM)))
	}

	grp.registerSetting(ordering, name, setting)
}

func (s *registeredGroup) registerSetting(ordering int, name string, setting Setting) {
	_, ok := s.settingsM[name]
	if ok {
		panic(fmt.Sprintf("setting %q was already registered in %v", name, maps.Keys(s.settingsM)))
	}

	s.settingsM[name] = setting
	s.settings = append(s.settings, registeredSetting{
		ordering:     ordering,
		internalName: name,
		setting:      setting,
	})
}
