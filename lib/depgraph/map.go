package depgraph

import "sort"

type ModuleReference struct {
	*Module
	VersionConstraint string
}

func (m *ModuleReference) Name() string { return m.Module.Name() }

type Reference interface {
	Name() string
}

type Dependencies struct {
	dependencyList []Reference
	dependencyMap  map[string]Reference
}

func NewDependencies() *Dependencies {
	return &Dependencies{
		dependencyList: []Reference{},
		dependencyMap:  map[string]Reference{},
	}
}

func (m *Dependencies) Len() int {
	return len(m.dependencyMap)
}

func (m *Dependencies) Copy() *Dependencies {
	newMap := &Dependencies{
		dependencyMap:  map[string]Reference{},
		dependencyList: make([]Reference, len(m.dependencyList)),
	}
	for _, dependency := range m.dependencyMap {
		newMap.dependencyMap[dependency.Name()] = dependency
	}
	copy(newMap.dependencyList, m.dependencyList)
	return newMap
}

func (m *Dependencies) Add(dependencyReference Reference) {
	if _, ok := m.dependencyMap[dependencyReference.Name()]; ok {
		return
	}

	m.dependencyMap[dependencyReference.Name()] = dependencyReference
	m.dependencyList = append(m.dependencyList, dependencyReference)
}

func (m *Dependencies) Get(name string) (Reference, bool) {
	dependency, ok := m.dependencyMap[name]
	return dependency, ok
}

func (m *Dependencies) Delete(name string) {
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

func (m *Dependencies) List() []Reference {
	sort.Slice(m.dependencyList, func(i int, j int) bool { return m.dependencyList[i].Name() < m.dependencyList[j].Name() })
	listCopy := make([]Reference, len(m.dependencyList))
	copy(listCopy, m.dependencyList)
	return listCopy
}
