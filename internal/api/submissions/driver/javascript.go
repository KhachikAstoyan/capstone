package driver

import (
	"fmt"
	"strings"
)

const jsHeader = `// __CAPSTONE_FUNC_RUNNER__
class TreeNode {
    constructor(val, left, right) {
        this.val   = val   === undefined ? 0    : val;
        this.left  = left  === undefined ? null : left;
        this.right = right === undefined ? null : right;
    }
}

class ListNode {
    constructor(val, next) {
        this.val  = val  === undefined ? 0    : val;
        this.next = next === undefined ? null : next;
    }
}

function deserializeTree(arr) {
    if (!arr || arr.length === 0 || arr[0] === null) return null;
    const root = new TreeNode(arr[0]);
    const queue = [root];
    let i = 1;
    while (queue.length > 0 && i < arr.length) {
        const node = queue.shift();
        if (i < arr.length && arr[i] !== null) {
            node.left = new TreeNode(arr[i]);
            queue.push(node.left);
        }
        i++;
        if (i < arr.length && arr[i] !== null) {
            node.right = new TreeNode(arr[i]);
            queue.push(node.right);
        }
        i++;
    }
    return root;
}

function serializeTree(root) {
    if (!root) return [];
    const result = [];
    const queue = [root];
    while (queue.length > 0) {
        const node = queue.shift();
        if (node) {
            result.push(node.val);
            queue.push(node.left);
            queue.push(node.right);
        } else {
            result.push(null);
        }
    }
    while (result.length > 0 && result[result.length - 1] === null) result.pop();
    return result;
}

function deserializeList(arr) {
    if (!arr || arr.length === 0) return null;
    let dummy = new ListNode(0);
    let cur = dummy;
    for (const v of arr) { cur.next = new ListNode(v); cur = cur.next; }
    return dummy.next;
}

function serializeList(head) {
    const result = [];
    while (head) { result.push(head.val); head = head.next; }
    return result;
}

function _deser(val, typ) {
    if (typ === 'TreeNode') return deserializeTree(val);
    if (typ === 'ListNode') return deserializeList(val);
    return val;
}

function _ser(val, typ) {
    if (typ === 'TreeNode') return serializeTree(val);
    if (typ === 'ListNode') return serializeList(val);
    return val;
}
`

func generateJavaScript(spec FunctionSpec) (header, footer string, err error) {
	var b strings.Builder
	b.WriteString("module.exports._runTest = function (_data) {\n")

	for _, p := range spec.Parameters {
		if p.Type == TypeTreeNode || p.Type == TypeListNode {
			b.WriteString(fmt.Sprintf(
				"    const _arg_%s = _deser(_data[%q], %q);\n",
				p.Name, p.Name, p.Type,
			))
		} else {
			b.WriteString(fmt.Sprintf(
				"    const _arg_%s = _data[%q];\n",
				p.Name, p.Name,
			))
		}
	}

	args := make([]string, len(spec.Parameters))
	for i, p := range spec.Parameters {
		args[i] = "_arg_" + p.Name
	}
	b.WriteString(fmt.Sprintf(
		"    const _result = %s(%s);\n",
		spec.FunctionName, strings.Join(args, ", "),
	))
	b.WriteString(fmt.Sprintf(
		"    return _ser(_result, %q);\n",
		spec.ReturnType,
	))
	b.WriteString("};\n")

	return jsHeader, b.String(), nil
}
