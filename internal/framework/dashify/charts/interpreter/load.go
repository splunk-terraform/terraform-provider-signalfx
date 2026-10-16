// Copyright Splunk, Inc.
// SPDX-License-Identifier: MPL-2.0

package interpreter

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"strings"
	"sync"

	"go.yaml.in/yaml/v3"

	"github.com/splunk-terraform/terraform-provider-signalfx/internal/framework/dashify/charts/definitions"
)

func New() (*Engine, error) { return Load(definitions.Files) }

var defaultEngine = sync.OnceValues(New)

// Default resolves the embedded chart definitions once for the provider.
// Invalid definitions return an error rather than panicking during startup.
func Default() (*Engine, error) { return defaultEngine() }

// Load resolves chart definitions from a definitions filesystem.
// Files at its root are charts; common files are loaded only through references.
func Load(files fs.FS) (*Engine, error) {
	names, err := fs.Glob(files, "*.yml")
	if err != nil {
		return nil, err
	}
	if len(names) == 0 {
		return nil, fmt.Errorf("no chart schemas found")
	}
	l := loader{files: files, cache: map[string]*yaml.Node{}}
	definitions := make([]Definition, 0, len(names))
	for _, name := range names {
		node, err := l.document(name, map[string]bool{})
		if err != nil {
			return nil, err
		}
		node, err = l.references(node, map[string]bool{})
		if err != nil {
			return nil, fmt.Errorf("%s: %w", name, err)
		}
		encoded, err := yaml.Marshal(node)
		if err != nil {
			return nil, err
		}
		var definition Definition
		decoder := yaml.NewDecoder(bytes.NewReader(encoded))
		decoder.KnownFields(true)
		if err := decoder.Decode(&definition); err != nil {
			return nil, fmt.Errorf("%s: %w", name, err)
		}
		definitions = append(definitions, definition)
	}
	return NewFromDefinitions(definitions)
}

type loader struct {
	files fs.FS
	cache map[string]*yaml.Node
}

func (l *loader) read(name string) (*yaml.Node, error) {
	if !fs.ValidPath(name) {
		return nil, fmt.Errorf("invalid schema path %q", name)
	}
	if node, exists := l.cache[name]; exists {
		return copyNode(node), nil
	}
	data, err := fs.ReadFile(l.files, name)
	if err != nil {
		return nil, err
	}
	var declared Definition
	decoder := yaml.NewDecoder(bytes.NewReader(data))
	decoder.KnownFields(true)
	if err := decoder.Decode(&declared); err != nil {
		return nil, fmt.Errorf("%s: %w", name, err)
	}
	var trailing any
	if err := decoder.Decode(&trailing); !errors.Is(err, io.EOF) {
		return nil, fmt.Errorf("%s: must contain exactly one YAML document", name)
	}
	if declared.Version != 1 {
		return nil, fmt.Errorf("%s: schema version must be 1", name)
	}
	var document yaml.Node
	if err := yaml.Unmarshal(data, &document); err != nil {
		return nil, fmt.Errorf("%s: %w", name, err)
	}
	if len(document.Content) != 1 || document.Content[0].Kind != yaml.MappingNode {
		return nil, fmt.Errorf("%s must be a YAML object", name)
	}
	if err := checkKeys(document.Content[0]); err != nil {
		return nil, fmt.Errorf("%s: %w", name, err)
	}
	l.cache[name] = document.Content[0]
	return copyNode(document.Content[0]), nil
}

func (l *loader) document(name string, active map[string]bool) (*yaml.Node, error) {
	if active[name] {
		return nil, fmt.Errorf("cyclic extends involving %s", name)
	}
	active[name] = true
	defer delete(active, name)
	node, err := l.read(name)
	if err != nil {
		return nil, err
	}
	var bases []string
	if inherited := field(node, "extends"); inherited != nil {
		if err := inherited.Decode(&bases); err != nil {
			return nil, err
		}
	}
	properties := &yaml.Node{Kind: yaml.MappingNode, Tag: "!!map"}
	for _, base := range bases {
		parent, err := l.document(base, active)
		if err != nil {
			return nil, err
		}
		if inherited := field(parent, "properties"); inherited != nil {
			properties = overlay(properties, inherited)
		}
	}
	if local := field(node, "properties"); local != nil {
		properties = overlay(properties, local)
	}
	result := copyNode(node)
	remove(result, "properties")
	result.Content = append(result.Content, &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: "properties"}, properties)
	remove(result, "extends")
	return result, nil
}

func (l *loader) references(node *yaml.Node, active map[string]bool) (*yaml.Node, error) {
	if reference := field(node, "ref"); reference != nil {
		key := reference.Value
		if active[key] {
			return nil, fmt.Errorf("cyclic ref %s", key)
		}
		parts := strings.Split(key, "#")
		if len(parts) != 2 || parts[1] == "" {
			return nil, fmt.Errorf("invalid ref %q", key)
		}
		active[key] = true
		document, err := l.document(parts[0], map[string]bool{})
		if err != nil {
			return nil, err
		}
		base := field(field(document, "properties"), parts[1])
		if base == nil {
			return nil, fmt.Errorf("ref %s is missing", key)
		}
		base, err = l.references(base, active)
		if err != nil {
			return nil, err
		}
		delete(active, key)
		node = overlay(base, node)
		remove(node, "ref")
	}
	for index, child := range node.Content {
		if node.Kind == yaml.MappingNode && index%2 == 0 {
			continue
		}
		resolved, err := l.references(child, active)
		if err != nil {
			return nil, err
		}
		node.Content[index] = resolved
	}
	return node, nil
}

// Overlay operates on authored YAML keys, so false, zero, empty collections,
// and explicit null replace inherited values without presence bookkeeping.
func overlay(base, local *yaml.Node) *yaml.Node {
	if base.Kind != yaml.MappingNode || local.Kind != yaml.MappingNode {
		return copyNode(local)
	}
	result := copyNode(base)
	for index := 0; index < len(local.Content); index += 2 {
		key, value := local.Content[index], local.Content[index+1]
		found := false
		for offset := 0; offset < len(result.Content); offset += 2 {
			if result.Content[offset].Value == key.Value {
				if key.Value == "mapping" {
					result.Content[offset+1] = overlay(result.Content[offset+1], value)
				} else {
					result.Content[offset+1] = copyNode(value)
				}
				found = true
				break
			}
		}
		if !found {
			result.Content = append(result.Content, copyNode(key), copyNode(value))
		}
	}
	return result
}
func field(node *yaml.Node, name string) *yaml.Node {
	if node == nil || node.Kind != yaml.MappingNode {
		return nil
	}
	for index := 0; index < len(node.Content); index += 2 {
		if node.Content[index].Value == name {
			return node.Content[index+1]
		}
	}
	return nil
}
func remove(node *yaml.Node, name string) {
	for index := 0; index < len(node.Content); index += 2 {
		if node.Content[index].Value == name {
			node.Content = append(node.Content[:index], node.Content[index+2:]...)
			return
		}
	}
}
func copyNode(node *yaml.Node) *yaml.Node {
	result := *node
	result.Content = make([]*yaml.Node, len(node.Content))
	for i, child := range node.Content {
		result.Content[i] = copyNode(child)
	}
	return &result
}
func checkKeys(node *yaml.Node) error {
	if node.Kind == yaml.AliasNode {
		return fmt.Errorf("YAML aliases are not supported")
	}
	seen := map[string]bool{}
	for index, child := range node.Content {
		if node.Kind == yaml.MappingNode && index%2 == 0 {
			if child.Kind != yaml.ScalarNode || child.Tag != "!!str" || seen[child.Value] {
				return fmt.Errorf("duplicate or non-string key %q", child.Value)
			}
			seen[child.Value] = true
		}
		if err := checkKeys(child); err != nil {
			return err
		}
	}
	return nil
}
