package depgraph

import "sort"

type Node interface {
	Name() string
	Predecessors() *Edges
	Successors() *Edges
}

type Edges struct {
	dependencyList []Node
	dependencyMap  map[string]Node
}

func NewEdges() Edges {
	return Edges{
		dependencyList: []Node{},
		dependencyMap:  map[string]Node{},
	}
}

func (m Edges) Len() int {
	return len(m.dependencyMap)
}

func (m Edges) Copy() Edges {
	newMap := Edges{
		dependencyMap:  map[string]Node{},
		dependencyList: make([]Node, len(m.dependencyList)),
	}
	for _, dependency := range m.dependencyMap {
		newMap.dependencyMap[dependency.Name()] = dependency
	}
	copy(newMap.dependencyList, m.dependencyList)
	return newMap
}

func (m *Edges) Add(dependencyReference Node) {
	if _, ok := m.dependencyMap[dependencyReference.Name()]; ok {
		return
	}

	m.dependencyMap[dependencyReference.Name()] = dependencyReference
	m.dependencyList = append(m.dependencyList, dependencyReference)
}

func (m Edges) Get(name string) (Node, bool) {
	dependency, ok := m.dependencyMap[name]
	return dependency, ok
}

func (m *Edges) Delete(name string) {
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

func (m Edges) List() []Node {
	sort.Slice(m.dependencyList, func(i int, j int) bool { return m.dependencyList[i].Name() < m.dependencyList[j].Name() })
	listCopy := make([]Node, len(m.dependencyList))
	copy(listCopy, m.dependencyList)
	return listCopy
}
