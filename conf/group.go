package conf

import (
	"sort"
	"strings"
)

// BelongsToGroup belongs to group or not.
func (c ServerConfig) BelongsToGroup(cf *Config, name string) bool {
	othersGroupName := cf.pickOthersGroupName()

	if len(c.Group) == 0 { // others
		return name == "" || name == othersGroupName
	}

	if name == "" || name == othersGroupName {
		return false
	}

	for _, g := range c.Group {
		if strings.HasPrefix(g, name) {
			return true
		}
	}

	return false
}

func (cf *Config) parseGroups() {
	cf.grouping = make(map[string]map[string]ServerConfig)

	for k, v := range cf.Server {
		if len(v.Group) == 0 {
			cf.addGroup("", k, v)
			continue
		}

		for _, group := range v.Group {
			cf.addGroup(group, k, v)
		}
	}
}

func (cf *Config) addGroup(group, name string, v ServerConfig) {
	if _, ok := cf.grouping[group]; ok {
		cf.grouping[group][name] = v
	} else {
		m := make(map[string]ServerConfig)
		m[name] = v
		cf.grouping[group] = m
	}
}

// FilterNamesByGroup  filter server names by group.
func (cf *Config) FilterNamesByGroup(group string, names []string) []string {
	if cf.Extra.DisableGrouping {
		return names
	}

	x := make([]string, 0, len(names))
	for _, name := range names {
		if sc := cf.Server[name]; sc.BelongsToGroup(cf, group) {
			x = append(x, name)
		}
	}

	return x
}

// GroupsNames get groups' names.
func (cf *Config) GroupsNames() []string {
	otherGroupName := cf.pickOthersGroupName()
	names := make([]string, 0)

	for k := range cf.grouping {
		if k == "" {
			k = otherGroupName
		}

		names = append(names, k)
	}

	sort.Strings(names)

	return names
}

func (cf *Config) pickOthersGroupName() string {
	otherGroupNames := []string{"others", "default", "else", "TDXX"} // no blanks allowed among the names
	otherGroupName := ""

	for _, groupName := range otherGroupNames {
		if _, ok := cf.grouping[groupName]; !ok {
			otherGroupName = groupName
			break
		}
	}

	return otherGroupName
}

// GetGrouping get grouping map.
func (cf *Config) GetGrouping() map[string]map[string]ServerConfig { return cf.grouping }
