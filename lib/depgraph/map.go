package depgraph

import "sort"

type NodeReference struct {
	*Dependency
	VersionConstraint string
}

type NodeMap struct {
	nodeList []*NodeReference
	nodeMap  map[string]*NodeReference
}

func NewNodeMap() *NodeMap {
	return &NodeMap{
		nodeList: []*NodeReference{},
		nodeMap:  map[string]*NodeReference{},
	}
}

func (m *NodeMap) Len() int {
	return len(m.nodeMap)
}

func (m *NodeMap) Copy() *NodeMap {
	newMap := &NodeMap{
		nodeMap:  map[string]*NodeReference{},
		nodeList: make([]*NodeReference, len(m.nodeList)),
	}
	for _, node := range m.nodeMap {
		newMap.nodeMap[node.Name()] = node
	}
	copy(newMap.nodeList, m.nodeList)
	return newMap
}

func (m *NodeMap) Add(nodeReference *NodeReference) {
	if _, ok := m.nodeMap[nodeReference.Name()]; ok {
		return
	}

	m.nodeMap[nodeReference.Name()] = nodeReference
	m.nodeList = append(m.nodeList, nodeReference)
}

func (m *NodeMap) Get(name string) (*NodeReference, bool) {
	node, ok := m.nodeMap[name]
	return node, ok
}

func (m *NodeMap) Delete(name string) {
	if _, ok := m.nodeMap[name]; !ok {
		return
	}
	delete(m.nodeMap, name)
	for idx := range m.nodeList {
		if m.nodeList[idx].Name() == name {
			m.nodeList = append(m.nodeList[:idx], m.nodeList[idx+1:]...)
			break
		}
	}
}

func (m *NodeMap) List() []*NodeReference {
	sort.Slice(m.nodeList, func(i int, j int) bool { return m.nodeList[i].Name() < m.nodeList[j].Name() })
	listCopy := make([]*NodeReference, len(m.nodeList))
	copy(listCopy, m.nodeList)
	return listCopy
}
