package groute

import (
	"fmt"
	"sort"
	"strings"
)

type Params struct {
	path        string
	node        *node
	params      map[string]int // param name => param position
	wildcardLen uint16
}

func (ps Params) IsZero() bool {
	return ps.node == nil
}

func (ps Params) Route() string {
	if ps.node != nil {
		return ps.node.route
	}
	return ""
}

func (ps Params) Get(name string) (string, bool) {
	if ps.node == nil {
		return "", false
	}
	if i, ok := ps.params[name]; ok {
		return ps.findParam(i)
	}
	return "", false
}

func (ps *Params) findParam(paramIndex int) (string, bool) {
	path := ps.path
	pathLen := len(path)
	currNode := ps.node
	currParamIndex := len(ps.params) - 1

	// Wildcard can be only in the final node.
	if ps.node.isWC {
		if currParamIndex == paramIndex {
			pathLen -= int(ps.wildcardLen)
			return path[pathLen:], true
		}

		currParamIndex--
		pathLen -= int(ps.wildcardLen)
		path = path[:pathLen]
	}

	for currNode != nil {
		if currNode.part[0] != ':' { // static node
			pathLen -= len(currNode.part)
			path = path[:pathLen]
			currNode = currNode.parent
			continue
		}

		i := strings.LastIndexByte(path, '/')
		if i == -1 {
			return "", false
		}
		pathLen = i + 1

		if currParamIndex == paramIndex {
			return path[pathLen:], true
		}

		currParamIndex--
		path = path[:pathLen]
		currNode = currNode.parent
	}

	return "", false
}

func (ps Params) ByName(name string) string {
	s, _ := ps.Get(name)
	return s
}

type node struct {
	route string
	part  string

	params  map[string]int // param name => param position
	handler interface{}    //

	parent *node
	colon  *node
	isWC   bool

	nodes []*node
	index struct {
		table   []uint8 // index table for the nodes: firstChar-minChar => node position
		minChar byte    // min char in the table
		maxChar byte    // max char in the table
	}
}

func (n *node) addRoute(route string, handler interface{}) error {
	parts, params := splitRoute(route)
	currNode := n

	for _, part := range parts {
		currNode = currNode.addPart(part)
	}

	if currNode.route == "" {
		currNode.route = route
	}
	n.indexNodes()

	if currNode.handler != nil {
		return fmt.Errorf("duplicate route, %v, %v", route, currNode.route)
	}

	currNode.params = params
	currNode.handler = handler
	return nil
}

func (n *node) addPart(part string) *node {
	if part == "*" {
		n.isWC = true
		return n
	}

	if part == ":" {
		if n.colon == nil {
			n.colon = &node{part: ":"}
		}
		return n.colon
	}

	for childNodeIndex, childNode := range n.nodes {
		if childNode.part[0] != part[0] {
			continue
		}

		// Check for a common prefix.

		for i, c := range []byte(part) {
			if i == len(childNode.part) {
				break
			}
			if c == childNode.part[i] {
				continue
			}

			// Create a node for the common prefix.

			childNode.part = childNode.part[i:]
			newNode := &node{part: part[i:]}

			n.nodes[childNodeIndex] = &node{
				part:  part[:i], // common prefix
				nodes: []*node{childNode, newNode},
			}

			return newNode
		}

		// Parts match completely.

		switch {
		case len(part) > len(childNode.part): // part is bigger
			part = part[len(childNode.part):]
			return childNode.addPart(part)

		case len(part) < len(childNode.part): // part is smaller
			childNode.part = childNode.part[len(part):]
			newNode := &node{part: part}
			newNode.nodes = []*node{childNode}
			n.nodes[childNodeIndex] = newNode
			return newNode

		default:
			return childNode // exact match
		}
	}

	node := &node{part: part}
	n.nodes = append(n.nodes, node)
	return node
}

func (n *node) findRoute(path string, params *Params) interface{} {
	if path == "" {
		return nil
	}
	path = path[1:] // strip leading "/"

	if path == "" {
		if n.handler != nil {
			params.path = path
			params.node = n
			params.params = n.params
			params.wildcardLen = 0
			return n.handler
		}
		return nil
	}

	node, handler, wildcardLen := n._findRoute(path)
	if handler != nil {
		params.path = path
		params.node = node
		params.params = node.params
		params.wildcardLen = uint16(wildcardLen)
		return handler
	}

	return nil
}

func (n *node) _findRoute(path string) (*node, interface{}, int) {
	var found *node

	if firstChar := path[0]; firstChar >= n.index.minChar && firstChar <= n.index.maxChar {
		if i := n.index.table[firstChar-n.index.minChar]; i != 0 {
			childNode := n.nodes[i-1]

			if childNode.part == path {
				if childNode.handler != nil {
					return childNode, childNode.handler, 0
				}
			} else {
				partLen := len(childNode.part)
				if strings.HasPrefix(path, childNode.part) {
					node, handler, wildcardLen := childNode._findRoute(path[partLen:])
					if handler != nil {
						return node, handler, wildcardLen
					}
					if node != nil {
						found = node
					}
				}
			}
		}
	}

	if n.colon != nil {
		if i := strings.IndexByte(path, '/'); i > 0 {
			node, handler, wildcardLen := n.colon._findRoute(path[i:])
			if handler != nil {
				return node, handler, wildcardLen
			}
		} else if n.colon.handler != nil {
			return n.colon, n.colon.handler, 0
		}
	}

	if n.isWC && n.handler != nil {
		return n, n.handler, len(path)
	}

	return found, nil, 0
}

func (n *node) indexNodes() {
	if len(n.nodes) > 0 {
		n._indexNodes()
	}

	if n.colon != nil {
		n.colon.parent = n
		n.colon.indexNodes()
	}
}

func (n *node) _indexNodes() {
	sort.Slice(n.nodes, func(i, j int) bool {
		return n.nodes[i].part[0] < n.nodes[j].part[0]
	})

	n.index.minChar = n.nodes[0].part[0]
	n.index.maxChar = n.nodes[len(n.nodes)-1].part[0]

	// Reset index.
	if size := int(n.index.maxChar - n.index.minChar + 1); len(n.index.table) != size {
		n.index.table = make([]uint8, size)
	} else {
		for i := range n.index.table {
			n.index.table[i] = 0
		}
	}

	// Index nodes by the first char in a part.
	for childNodeIndex, childNode := range n.nodes {
		childNode.parent = n
		childNode.indexNodes()

		firstChar := childNode.part[0] - n.index.minChar
		n.index.table[firstChar] = uint8(childNodeIndex + 1)
	}
}

//------------------------------------------------------------------------------

type routeParser struct {
	segments []string
	i        int

	acc   []string
	parts []string
}

func (p *routeParser) valid() bool {
	return p.i < len(p.segments)
}

func (p *routeParser) next() string {
	s := p.segments[p.i]
	p.i++
	return s
}

func (p *routeParser) accumulate(s string) {
	p.acc = append(p.acc, s)
}

func (p *routeParser) finalizePart(withSlash bool) {
	if part := join(p.acc, withSlash); part != "" {
		p.parts = append(p.parts, part)
	}
	p.acc = p.acc[:0]

	if p.valid() {
		p.acc = append(p.acc, "")
	}
}

func join(ss []string, withSlash bool) string {
	if len(ss) == 0 {
		return ""
	}
	s := strings.Join(ss, "/")
	if withSlash {
		return s + "/"
	}
	return s
}

func splitRoute(route string) (_ []string, _ map[string]int) {
	if route == "" || route[0] != '/' {
		panic(fmt.Errorf("invalid route: %q", route))
	}

	if route == "/" {
		return []string{}, nil
	}
	route = route[1:] // trim first "/"

	ss := strings.Split(route, "/")
	if len(ss) == 0 {
		panic(fmt.Errorf("invalid route: %q", route))
	}

	p := routeParser{
		segments: ss,
	}
	var params []string

	for p.valid() {
		segment := p.next()

		if segment == "" {
			p.accumulate("")
			continue
		}

		switch firstChar := segment[0]; firstChar {
		case ':':
			p.finalizePart(true)
			p.parts = append(p.parts, ":")
			params = append(params, segment[1:])
		case '*':
			p.finalizePart(true)
			p.parts = append(p.parts, "*")
			params = append(params, segment[1:])
		default:
			p.accumulate(segment)
		}
	}

	p.finalizePart(false)

	if len(params) > 0 {
		return p.parts, paramMap(route, params)
	}
	return p.parts, nil
}

func paramMap(route string, params []string) map[string]int {
	m := make(map[string]int, len(params))
	for i, param := range params {
		if param == "" {
			panic(fmt.Errorf("param must have a name: %q", route))
		}
		m[param] = i
	}
	return m
}
