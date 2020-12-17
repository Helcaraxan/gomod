package depgraph

import "sort"

type ModuleReference struct {
	*Module
	VersionConstraint string
}

type ModuleDependencies struct {
	dependencyList []*ModuleReference
	dependencyMap  map[string]*ModuleReference
}

func NewModuleDependencies() *ModuleDependencies {
	return &ModuleDependencies{
		dependencyList: []*ModuleReference{},
		dependencyMap:  map[string]*ModuleReference{},
	}
}

func (m *ModuleDependencies) Len() int {
	return len(m.dependencyMap)
}

func (m *ModuleDependencies) Copy() *ModuleDependencies {
	newMap := &ModuleDependencies{
		dependencyMap:  map[string]*ModuleReference{},
		dependencyList: make([]*ModuleReference, len(m.dependencyList)),
	}
	for _, dependency := range m.dependencyMap {
		newMap.dependencyMap[dependency.Name()] = dependency
	}
	copy(newMap.dependencyList, m.dependencyList)
	return newMap
}

func (m *ModuleDependencies) Add(dependencyReference *ModuleReference) {
	if _, ok := m.dependencyMap[dependencyReference.Name()]; ok {
		return
	}

	m.dependencyMap[dependencyReference.Name()] = dependencyReference
	m.dependencyList = append(m.dependencyList, dependencyReference)
}

func (m *ModuleDependencies) Get(name string) (*ModuleReference, bool) {
	dependency, ok := m.dependencyMap[name]
	return dependency, ok
}

func (m *ModuleDependencies) Delete(name string) {
	if _, ok := m.dependencyMap[name]; !ok {
		return
	}
	delete(m.dependencyMap, name)
	for idx := range m.dependencyList {
		if m.dependencyList[idx].Name() == name {
			m.dependencyList = append(m.dependencyList[:idx], m.dependencyList[idx+1:]...)
			break
		}
	}
}

func (m *ModuleDependencies) List() []*ModuleReference {
	sort.Slice(m.dependencyList, func(i int, j int) bool { return m.dependencyList[i].Name() < m.dependencyList[j].Name() })
	listCopy := make([]*ModuleReference, len(m.dependencyList))
	copy(listCopy, m.dependencyList)
	return listCopy
}
