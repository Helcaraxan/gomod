package graph

import (
	"reflect"
	"sort"
	"strings"
)

type Node interface {
	Name() string
	Hash() string
	String() string

	Predecessors() *NodeRefs
	Successors() *NodeRefs

	Parent() Node
	Children() *NodeRefs
}

func nodeIsNil(n Node) bool {
	return n == nil || reflect.ValueOf(n).IsNil()
}

func nodeDepth(n Node) int {
	depth := -1
	for !nodeIsNil(n) {
		depth++
		n = n.Parent()
	}
	return depth
}

type NodeRefs struct {
	nodeList []Node
	nodeMap  map[string]Node
	weights  map[string]int
}

func NewNodeRefs() NodeRefs {
	return NodeRefs{
		nodeList: []Node{},
		nodeMap:  map[string]Node{},
		weights:  map[string]int{},
	}
}

func (n NodeRefs) String() string {
	var ns []string
	for _, m := range n.nodeList {
		ns = append(ns, m.Name())
	}
	return strings.Join(ns, ", ")
}

func (n NodeRefs) Len() int {
	return len(n.nodeMap)
}

func (n *NodeRefs) Add(node Node) {
	if nodeIsNil(node) {
		return
	}

	h := node.Hash()
	n.weights[h]++
	if _, ok := n.nodeMap[h]; !ok {
		n.nodeMap[h] = node
		n.nodeList = append(n.nodeList, node)
	}
}

func (n NodeRefs) Get(hash string) (Node, int) {
	return n.nodeMap[hash], n.weights[hash]
}

func (n *NodeRefs) Delete(hash string) {
	if n == nil {
		return
	}

	if _, ok := n.nodeMap[hash]; !ok {
		return
	}

	n.weights[hash]--
	if n.weights[hash] > 0 {
		return
	}

	delete(n.nodeMap, hash)
	delete(n.weights, hash)

	for idx := range n.nodeList {
		if n.nodeList[idx].Hash() == hash {
			n.nodeList = append(n.nodeList[:idx], n.nodeList[idx+1:]...)
			break
		}
	}
}

func (n *NodeRefs) Wipe(hash string) {
	if n == nil {
		return
	}

	if _, ok := n.nodeMap[hash]; !ok {
		return
	}

	delete(n.nodeMap, hash)
	delete(n.weights, hash)

	for idx := range n.nodeList {
		if n.nodeList[idx].Hash() == hash {
			n.nodeList = append(n.nodeList[:idx], n.nodeList[idx+1:]...)
			break
		}
	}
}

func (n NodeRefs) List() []Node {
	sort.Slice(n.nodeList, func(i int, j int) bool { return n.nodeList[i].Name() < n.nodeList[j].Name() })
	listCopy := make([]Node, len(n.nodeList))
	copy(listCopy, n.nodeList)
	return listCopy
}
