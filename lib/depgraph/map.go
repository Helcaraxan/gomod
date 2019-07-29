package depgraph

import "sort"

type NodeMap struct {
	nodeList []*Node
	nodeMap  map[string]*Node
}

func NewNodeMap() *NodeMap {
	return &NodeMap{
		nodeList: []*Node{},
		nodeMap:  map[string]*Node{},
	}
}

func (m *NodeMap) Len() int {
	return len(m.nodeMap)
}

func (m *NodeMap) Copy() *NodeMap {
	newMap := &NodeMap{
		nodeMap:  map[string]*Node{},
		nodeList: make([]*Node, len(m.nodeList)),
	}
	for _, node := range m.nodeMap {
		newMap.nodeMap[node.Name()] = node
	}
	copy(newMap.nodeList, m.nodeList)
	return newMap
}

func (m *NodeMap) Add(node *Node) {
	if _, ok := m.nodeMap[node.Name()]; ok {
		return
	}

	m.nodeMap[node.Name()] = node
	m.nodeList = append(m.nodeList, node)
}

func (m *NodeMap) Get(name string) (*Node, bool) {
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

func (m *NodeMap) List() []*Node {
	sort.Slice(m.nodeList, func(i int, j int) bool { return m.nodeList[i].Name() < m.nodeList[j].Name() })
	listCopy := make([]*Node, len(m.nodeList))
	copy(listCopy, m.nodeList)
	return listCopy
}
