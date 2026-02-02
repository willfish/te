package parsing

import (
	"encoding/json"
	"fmt"
	"io"
	"regexp"

	"github.com/orisano/gosax"
	"github.com/willfish/te/internal/store"
)

type Node map[string]interface{}

func Parse(f io.Reader, s *store.Store) error {
	targetDepth := 4
	inTarget := false
	extraContent := regexp.MustCompile(`^\n\s+`)
	contentKey := "__content__"
	stack := []Node{}
	node := Node{}

	depth := 0
	r := gosax.NewReader(f)
	for {
		e, err := r.Event()
		if err != nil {
			return fmt.Errorf("reading XML event: %w", err)
		}
		if e.Type() == gosax.EventEOF {
			break
		}
		switch e.Type() {
		case gosax.EventStart:
			depth++

			if depth == targetDepth {
				inTarget = true
			}
			if inTarget {
				if len(stack) > 0 {
					lastNode := stack[len(stack)-1]
					delete(lastNode, contentKey)
				}
				node = Node{contentKey: ""}
				stack = append(stack, node)
			}
		case gosax.EventText:
			if inTarget {
				if !extraContent.Match(e.Bytes) {
					value := string(e.Bytes)
					if len(value) > 0 {
						if current, ok := node[contentKey].(string); ok {
							node[contentKey] = current + value
						} else {
							node[contentKey] = value
						}
					}
				}
			}
		case gosax.EventEnd:
			key := string(e.Bytes)
			if len(key) >= 3 {
				key = key[2 : len(key)-1]
			}

			if depth == targetDepth {
				n := stack[len(stack)-1]
				targetHandler(n)

				hjid := fmt.Sprintf("%v", n["hjid"])
				jsonData, err := json.Marshal(n)
				if err != nil {
					return fmt.Errorf("marshalling element %s: %w", hjid, err)
				}

				if err := s.InsertElement(hjid, key, string(jsonData)); err != nil {
					return fmt.Errorf("inserting element %s: %w", hjid, err)
				}

				stack = stack[:len(stack)-1]
				inTarget = false
			}
			depth--

			if inTarget {
				child := stack[len(stack)-1]
				node := stack[len(stack)-2]
				stack = stack[:len(stack)-1]

				switch v := node[key].(type) {
				case []Node:
					node[key] = append(v, child)
				case Node:
					node[key] = []Node{v, child}
				default:
					if len(child) == 1 && child[contentKey] != "" {
						node[key] = child[contentKey]
					} else {
						node[key] = child
					}
				}
			}
		}
	}

	if err := s.Flush(); err != nil {
		return fmt.Errorf("flushing store: %w", err)
	}

	return nil
}

func deepFlatten(n Node, prefix string) Node {
	flattened := Node{}
	for k, v := range n {
		if v == nil {
			delete(n, k)
			continue
		}

		switch v := v.(type) {
		case Node:
			for kk, vv := range deepFlatten(v, fmt.Sprintf("%s.%s", prefix, k)) {
				flattened[kk] = vv
			}
		case []Node:
			for i, n := range v {
				for kk, vv := range deepFlatten(n, fmt.Sprintf("%s.%s[%d]", prefix, k, i)) {
					flattened[kk] = vv
				}
			}
		default:
			flattened[fmt.Sprintf("%s.%s", prefix, k)] = v
		}
	}
	return flattened
}

func targetHandler(n Node) Node {
	for k, v := range n {
		if v == nil {
			continue
		}
		switch v := v.(type) {
		case Node:
			if v["metainfo"] == nil {
				flattened := deepFlatten(v, k)
				for kk, vv := range flattened {
					n[kk] = vv
				}
				delete(n, k)
			} else {
				n[k] = targetHandler(v)
			}
		case []Node:
			nodes := []Node{}
			for _, n := range v {
				nodes = append(nodes, targetHandler(n))
			}
			n[k] = nodes
		}
	}

	return n
}
