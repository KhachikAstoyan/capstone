package driver

import (
	"fmt"
	"strings"
)

const goHeader = `package main

import (
	"encoding/json"
	"fmt"
	"os"
)

type TreeNode struct {
	Val   int
	Left  *TreeNode
	Right *TreeNode
}

type ListNode struct {
	Val  int
	Next *ListNode
}

func deserializeTree(raw json.RawMessage) *TreeNode {
	var arr []interface{}
	if err := json.Unmarshal(raw, &arr); err != nil || len(arr) == 0 {
		return nil
	}
	toInt := func(v interface{}) (int, bool) {
		f, ok := v.(float64)
		return int(f), ok
	}
	val, ok := toInt(arr[0])
	if !ok {
		return nil
	}
	root := &TreeNode{Val: val}
	queue := []*TreeNode{root}
	i := 1
	for len(queue) > 0 && i < len(arr) {
		node := queue[0]
		queue = queue[1:]
		if i < len(arr) && arr[i] != nil {
			v, _ := toInt(arr[i])
			node.Left = &TreeNode{Val: v}
			queue = append(queue, node.Left)
		}
		i++
		if i < len(arr) && arr[i] != nil {
			v, _ := toInt(arr[i])
			node.Right = &TreeNode{Val: v}
			queue = append(queue, node.Right)
		}
		i++
	}
	return root
}

func serializeTree(root *TreeNode) interface{} {
	if root == nil {
		return []interface{}{}
	}
	var result []interface{}
	queue := []*TreeNode{root}
	for len(queue) > 0 {
		node := queue[0]
		queue = queue[1:]
		if node != nil {
			result = append(result, node.Val)
			queue = append(queue, node.Left, node.Right)
		} else {
			result = append(result, nil)
		}
	}
	for len(result) > 0 && result[len(result)-1] == nil {
		result = result[:len(result)-1]
	}
	return result
}

func deserializeList(arr []int) *ListNode {
	if len(arr) == 0 {
		return nil
	}
	head := &ListNode{Val: arr[0]}
	cur := head
	for _, v := range arr[1:] {
		cur.Next = &ListNode{Val: v}
		cur = cur.Next
	}
	return head
}

func serializeList(head *ListNode) []int {
	var result []int
	for head != nil {
		result = append(result, head.Val)
		head = head.Next
	}
	return result
}
`

func generateGo(spec FunctionSpec) (header, footer string, err error) {
	var b strings.Builder
	b.WriteString("func main() {\n")

	// Build anonymous input struct
	b.WriteString("\tvar _in struct {\n")
	for _, p := range spec.Parameters {
		goType, _ := goFieldType(p.Type)
		fieldName := upperFirst(p.Name)
		b.WriteString(fmt.Sprintf("\t\t%s %s `json:%q`\n", fieldName, goType, p.Name))
	}
	b.WriteString("\t}\n")

	b.WriteString("\tif err := json.NewDecoder(os.Stdin).Decode(&_in); err != nil {\n")
	b.WriteString("\t\tfmt.Fprintln(os.Stderr, err)\n")
	b.WriteString("\t\tos.Exit(1)\n")
	b.WriteString("\t}\n")

	// Deserialize complex types
	for _, p := range spec.Parameters {
		_, isRaw := goFieldType(p.Type)
		if !isRaw {
			continue
		}
		fieldName := upperFirst(p.Name)
		argName := "_arg_" + p.Name
		switch p.Type {
		case TypeTreeNode:
			b.WriteString(fmt.Sprintf("\t%s := deserializeTree(_in.%s)\n", argName, fieldName))
		case TypeListNode:
			b.WriteString(fmt.Sprintf("\tvar _raw_%s []int\n", p.Name))
			b.WriteString(fmt.Sprintf("\t_ = json.Unmarshal(_in.%s, &_raw_%s)\n", fieldName, p.Name))
			b.WriteString(fmt.Sprintf("\t%s := deserializeList(_raw_%s)\n", argName, p.Name))
		}
	}

	// Build call arguments
	args := make([]string, len(spec.Parameters))
	for i, p := range spec.Parameters {
		_, isRaw := goFieldType(p.Type)
		if isRaw {
			args[i] = "_arg_" + p.Name
		} else {
			args[i] = "_in." + upperFirst(p.Name)
		}
	}

	b.WriteString(fmt.Sprintf("\t_result := %s(%s)\n", spec.FunctionName, strings.Join(args, ", ")))

	// Write result to a file so user's fmt.Println output stays on stdout cleanly.
	switch spec.ReturnType {
	case TypeTreeNode:
		b.WriteString("\t_resultBytes, _ := json.Marshal(serializeTree(_result))\n")
	case TypeListNode:
		b.WriteString("\t_resultBytes, _ := json.Marshal(serializeList(_result))\n")
	default:
		b.WriteString("\t_resultBytes, _ := json.Marshal(_result)\n")
	}
	b.WriteString("\t_ = os.WriteFile(\"/tmp/capstone_result\", _resultBytes, 0o644)\n")

	b.WriteString("}\n")
	return goHeader, b.String(), nil
}

// goFieldType maps a ParamType to the Go struct field type.
// isRaw=true means the field is json.RawMessage and needs manual deserialization.
func goFieldType(t ParamType) (goType string, isRaw bool) {
	switch t {
	case TypeInt:
		return "int", false
	case TypeFloat:
		return "float64", false
	case TypeString:
		return "string", false
	case TypeBool:
		return "bool", false
	case TypeIntArray:
		return "[]int", false
	case TypeFloatArray:
		return "[]float64", false
	case TypeStringArray:
		return "[]string", false
	case TypeBoolArray:
		return "[]bool", false
	case TypeIntMatrix:
		return "[][]int", false
	case TypeStringMatrix:
		return "[][]string", false
	case TypeListNode:
		return "json.RawMessage", true
	case TypeTreeNode:
		return "json.RawMessage", true
	default:
		return "interface{}", false
	}
}

func upperFirst(s string) string {
	if s == "" {
		return s
	}
	return strings.ToUpper(s[:1]) + s[1:]
}

// extractGoImports scans user source and returns:
//   - importPaths: each user-declared import path (deduped, no quotes)
//   - cleaned: source with all import declarations removed
//
// Handles three forms: `import "x"`, `import "x"; import "y"`,
// and `import ( "x"; "y" )`. Also strips a leading `package main`
// line if present (the wrapper supplies its own package clause).
func extractGoImports(src string) (importPaths []string, cleaned string) {
	seen := make(map[string]bool)
	add := func(p string) {
		p = strings.TrimSpace(p)
		if p == "" || seen[p] {
			return
		}
		seen[p] = true
		importPaths = append(importPaths, p)
	}

	lines := strings.Split(src, "\n")
	out := make([]string, 0, len(lines))
	inImportBlock := false

	for i := 0; i < len(lines); i++ {
		line := lines[i]
		trimmed := strings.TrimSpace(line)

		if inImportBlock {
			if trimmed == ")" {
				inImportBlock = false
				continue
			}
			if trimmed == "" || strings.HasPrefix(trimmed, "//") {
				continue
			}
			// Form: `alias "path"` or `"path"` — extract the quoted path.
			if q := extractQuoted(trimmed); q != "" {
				add(q)
			}
			continue
		}

		if strings.HasPrefix(trimmed, "package ") {
			// Drop the user's package clause (wrapper provides one).
			continue
		}

		if trimmed == "import (" || trimmed == "import(" {
			inImportBlock = true
			continue
		}

		if strings.HasPrefix(trimmed, "import ") {
			// Single-line: `import "path"` or `import alias "path"`.
			if q := extractQuoted(trimmed); q != "" {
				add(q)
			}
			continue
		}

		out = append(out, line)
	}

	return importPaths, strings.Join(out, "\n")
}

// extractQuoted returns the contents of the first double-quoted substring,
// or "" if none. Used to pull a single import path out of a line.
func extractQuoted(s string) string {
	a := strings.Index(s, `"`)
	if a < 0 {
		return ""
	}
	b := strings.Index(s[a+1:], `"`)
	if b < 0 {
		return ""
	}
	return s[a+1 : a+1+b]
}

// injectGoImports merges userImports into the header's import block.
// Skips imports already declared in the header to avoid duplicates.
func injectGoImports(header string, userImports []string) string {
	if len(userImports) == 0 {
		return header
	}
	const closeMarker = ")"
	idx := strings.Index(header, "import (")
	if idx < 0 {
		return header
	}
	endRel := strings.Index(header[idx:], closeMarker)
	if endRel < 0 {
		return header
	}
	end := idx + endRel
	block := header[idx:end]
	var add strings.Builder
	for _, p := range userImports {
		// Already-present check: search for the quoted path in the existing block.
		if strings.Contains(block, `"`+p+`"`) {
			continue
		}
		add.WriteString("\t\"")
		add.WriteString(p)
		add.WriteString("\"\n")
	}
	if add.Len() == 0 {
		return header
	}
	return header[:end] + add.String() + header[end:]
}
