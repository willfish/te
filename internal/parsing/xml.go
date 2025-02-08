package parsing

import (
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"regexp"

	"github.com/orisano/gosax"
	"go.mongodb.org/mongo-driver/bson"
)

type Node map[string]interface{}
type Primaries map[string]int

var primaries = Primaries{}

func Parse(f io.Reader, filename string) {
	base := "$HOME/.cache/te/" + filepath.Base(filename)
	filename = os.ExpandEnv(filename)
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
			log.Fatal(err)
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
				primaries[key]++
				targetHandler(n)
				path := []string{base, key}
				writeNode(n, path)
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

	primaryFile, err := os.Create(filepath.Join(base, "primaries.bson"))
	defer primaryFile.Close()
	if err != nil {
		log.Fatalf("Error creating file: %v - do you have write permissions", err)
	}

	if data, err := bson.Marshal(primaries); err != nil {
		log.Fatalf("Error marshalling primaries: %v", err)
	} else {
		_, err = primaryFile.Write(data)
		if err != nil {
			log.Fatalf("Error writing to file: %v", err)
		}
	}

}

func encodeNode(n Node) ([]byte, error) {
	return bson.Marshal(n)
}

func decodeNode(data []byte) (Node, error) {
	var n Node
	err := bson.Unmarshal(data, &n)
	return n, err
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

func writeNode(n Node, path []string) {
	hjid := n["hjid"]
	newFile := filepath.Join(path...)
	newFile = os.ExpandEnv(newFile)

	err := os.MkdirAll(newFile, os.ModePerm)
	if err != nil {
		log.Fatalf("Error creating path: %v", err)
	}
	path = append(path, fmt.Sprintf("%v", hjid)+".bson")

	filePath := filepath.Join(path...)
	filePath = os.ExpandEnv(filePath)

	bsonFile, err := os.Create(filePath)
	defer bsonFile.Close()
	if err != nil {
		log.Fatalf("Error creating file: %v - do you have write permissions", err)
	}

	data, err := encodeNode(n)
	if err != nil {
		log.Printf("Error marshalling node: %v", err)
		return
	}
	_, err = bsonFile.WriteString(string(data))
}
