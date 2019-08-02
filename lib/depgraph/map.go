package depgraph

import "sort"

type DependencyReference struct {
	*Dependency
	VersionConstraint string
}

type DependencyMap struct {
	dependencyList []*DependencyReference
	dependencyMap  map[string]*DependencyReference
}

func NewDependencyMap() *DependencyMap {
	return &DependencyMap{
		dependencyList: []*DependencyReference{},
		dependencyMap:  map[string]*DependencyReference{},
	}
}

func (m *DependencyMap) Len() int {
	return len(m.dependencyMap)
}

func (m *DependencyMap) Copy() *DependencyMap {
	newMap := &DependencyMap{
		dependencyMap:  map[string]*DependencyReference{},
		dependencyList: make([]*DependencyReference, len(m.dependencyList)),
	}
	for _, dependency := range m.dependencyMap {
		newMap.dependencyMap[dependency.Name()] = dependency
	}
	copy(newMap.dependencyList, m.dependencyList)
	return newMap
}

func (m *DependencyMap) Add(dependencyReference *DependencyReference) {
	if _, ok := m.dependencyMap[dependencyReference.Name()]; ok {
		return
	}

	m.dependencyMap[dependencyReference.Name()] = dependencyReference
	m.dependencyList = append(m.dependencyList, dependencyReference)
}

func (m *DependencyMap) Get(name string) (*DependencyReference, bool) {
	dependency, ok := m.dependencyMap[name]
	return dependency, ok
}

func (m *DependencyMap) Delete(name string) {
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

func (m *DependencyMap) List() []*DependencyReference {
	sort.Slice(m.dependencyList, func(i int, j int) bool { return m.dependencyList[i].Name() < m.dependencyList[j].Name() })
	listCopy := make([]*DependencyReference, len(m.dependencyList))
	copy(listCopy, m.dependencyList)
	return listCopy
}
