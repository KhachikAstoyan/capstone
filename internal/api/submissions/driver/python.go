package driver

import (
	"fmt"
	"strings"
)

const pythonHeader = `# __CAPSTONE_FUNC_RUNNER__
from typing import Optional, List


class TreeNode:
    def __init__(self, val=0, left=None, right=None):
        self.val = val
        self.left = left
        self.right = right


class ListNode:
    def __init__(self, val=0, next=None):
        self.val = val
        self.next = next


def _deserialize_tree(arr):
    if not arr:
        return None
    root = TreeNode(arr[0])
    queue = [root]
    i = 1
    while queue and i < len(arr):
        node = queue.pop(0)
        if i < len(arr) and arr[i] is not None:
            node.left = TreeNode(arr[i])
            queue.append(node.left)
        i += 1
        if i < len(arr) and arr[i] is not None:
            node.right = TreeNode(arr[i])
            queue.append(node.right)
        i += 1
    return root


def _serialize_tree(root):
    if not root:
        return []
    result, queue = [], [root]
    while queue:
        node = queue.pop(0)
        if node:
            result.append(node.val)
            queue.append(node.left)
            queue.append(node.right)
        else:
            result.append(None)
    while result and result[-1] is None:
        result.pop()
    return result


def _deserialize_list(arr):
    if not arr:
        return None
    head = ListNode(arr[0])
    cur = head
    for v in arr[1:]:
        cur.next = ListNode(v)
        cur = cur.next
    return head


def _serialize_list(head):
    result = []
    while head:
        result.append(head.val)
        head = head.next
    return result


def _deser(val, typ):
    if typ == 'TreeNode':
        return _deserialize_tree(val)
    if typ == 'ListNode':
        return _deserialize_list(val)
    return val


def _ser(val, typ):
    if typ == 'TreeNode':
        return _serialize_tree(val)
    if typ == 'ListNode':
        return _serialize_list(val)
    return val
`

func generatePython(spec FunctionSpec) (header, footer string, err error) {
	var b strings.Builder
	b.WriteString("import json as _json\n\n")
	b.WriteString("def _run_test(_data):\n")

	for _, p := range spec.Parameters {
		b.WriteString(fmt.Sprintf(
			"    _arg_%s = _deser(_data[%q], %q)\n",
			p.Name, p.Name, p.Type,
		))
	}

	args := make([]string, len(spec.Parameters))
	for i, p := range spec.Parameters {
		args[i] = "_arg_" + p.Name
	}
	b.WriteString(fmt.Sprintf(
		"    _result = %s(%s)\n",
		spec.FunctionName, strings.Join(args, ", "),
	))
	b.WriteString(fmt.Sprintf(
		"    return _ser(_result, %q)\n",
		spec.ReturnType,
	))

	return pythonHeader, b.String(), nil
}
