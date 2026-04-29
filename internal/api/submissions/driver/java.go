package driver

import (
	"fmt"
	"strings"
)

const javaHeader = `// __CAPSTONE_FUNC_RUNNER__
import java.util.*;
import java.io.*;
import java.nio.file.*;
import com.fasterxml.jackson.databind.*;
import com.fasterxml.jackson.databind.node.*;

public class Main {
    static class TreeNode {
        int val;
        TreeNode left;
        TreeNode right;
        TreeNode() {}
        TreeNode(int val) { this.val = val; }
        TreeNode(int val, TreeNode left, TreeNode right) {
            this.val = val; this.left = left; this.right = right;
        }
    }

    static class ListNode {
        int val;
        ListNode next;
        ListNode() {}
        ListNode(int val) { this.val = val; }
        ListNode(int val, ListNode next) { this.val = val; this.next = next; }
    }

    static TreeNode deserializeTree(JsonNode arr) {
        if (arr == null || arr.isNull() || !arr.isArray() || arr.size() == 0) return null;
        if (arr.get(0).isNull()) return null;
        TreeNode root = new TreeNode(arr.get(0).asInt());
        Deque<TreeNode> queue = new ArrayDeque<>();
        queue.add(root);
        int i = 1;
        while (!queue.isEmpty() && i < arr.size()) {
            TreeNode node = queue.pollFirst();
            if (i < arr.size() && !arr.get(i).isNull()) {
                node.left = new TreeNode(arr.get(i).asInt());
                queue.add(node.left);
            }
            i++;
            if (i < arr.size() && !arr.get(i).isNull()) {
                node.right = new TreeNode(arr.get(i).asInt());
                queue.add(node.right);
            }
            i++;
        }
        return root;
    }

    static List<Object> serializeTree(TreeNode root) {
        List<Object> result = new ArrayList<>();
        if (root == null) return result;
        Deque<TreeNode> queue = new ArrayDeque<>();
        queue.add(root);
        while (!queue.isEmpty()) {
            TreeNode node = queue.pollFirst();
            if (node != null) {
                result.add(node.val);
                queue.add(node.left);
                queue.add(node.right);
            } else {
                result.add(null);
            }
        }
        while (!result.isEmpty() && result.get(result.size() - 1) == null) {
            result.remove(result.size() - 1);
        }
        return result;
    }

    static ListNode deserializeList(JsonNode arr) {
        if (arr == null || !arr.isArray() || arr.size() == 0) return null;
        ListNode head = new ListNode(arr.get(0).asInt());
        ListNode cur = head;
        for (int i = 1; i < arr.size(); i++) {
            cur.next = new ListNode(arr.get(i).asInt());
            cur = cur.next;
        }
        return head;
    }

    static List<Integer> serializeList(ListNode head) {
        List<Integer> result = new ArrayList<>();
        while (head != null) {
            result.add(head.val);
            head = head.next;
        }
        return result;
    }

    static class Solution {
`

func generateJava(spec FunctionSpec) (header, footer string, err error) {
	var b strings.Builder
	b.WriteString("    }\n\n")
	b.WriteString("    public static void main(String[] args) throws Exception {\n")
	b.WriteString("        ObjectMapper mapper = new ObjectMapper();\n")
	b.WriteString("        JsonNode _in = mapper.readTree(System.in);\n")
	b.WriteString("        Solution _sol = new Solution();\n")

	for _, p := range spec.Parameters {
		expr, derr := javaArgDeser(p)
		if derr != nil {
			return "", "", derr
		}
		b.WriteString(fmt.Sprintf("        %s _arg_%s = %s;\n", javaTypeName(p.Type), p.Name, expr))
	}

	args := make([]string, len(spec.Parameters))
	for i, p := range spec.Parameters {
		args[i] = "_arg_" + p.Name
	}
	b.WriteString(fmt.Sprintf("        %s _result = _sol.%s(%s);\n",
		javaTypeName(spec.ReturnType), spec.FunctionName, strings.Join(args, ", ")))

	switch spec.ReturnType {
	case TypeTreeNode:
		b.WriteString("        Object _ser = serializeTree(_result);\n")
	case TypeListNode:
		b.WriteString("        Object _ser = serializeList(_result);\n")
	default:
		b.WriteString("        Object _ser = _result;\n")
	}
	b.WriteString("        Files.write(Paths.get(\"/tmp/capstone_result\"), mapper.writeValueAsBytes(_ser));\n")
	b.WriteString("    }\n")
	b.WriteString("}\n")
	return javaHeader, b.String(), nil
}

func javaTypeName(t ParamType) string {
	switch t {
	case TypeInt:
		return "int"
	case TypeFloat:
		return "double"
	case TypeString:
		return "String"
	case TypeBool:
		return "boolean"
	case TypeIntArray:
		return "int[]"
	case TypeFloatArray:
		return "double[]"
	case TypeStringArray:
		return "String[]"
	case TypeBoolArray:
		return "boolean[]"
	case TypeIntMatrix:
		return "int[][]"
	case TypeStringMatrix:
		return "String[][]"
	case TypeListNode:
		return "ListNode"
	case TypeTreeNode:
		return "TreeNode"
	}
	return "Object"
}

func javaArgDeser(p Parameter) (string, error) {
	switch p.Type {
	case TypeInt:
		return fmt.Sprintf("_in.get(%q).asInt()", p.Name), nil
	case TypeFloat:
		return fmt.Sprintf("_in.get(%q).asDouble()", p.Name), nil
	case TypeString:
		return fmt.Sprintf("_in.get(%q).asText()", p.Name), nil
	case TypeBool:
		return fmt.Sprintf("_in.get(%q).asBoolean()", p.Name), nil
	case TypeIntArray:
		return fmt.Sprintf("mapper.treeToValue(_in.get(%q), int[].class)", p.Name), nil
	case TypeFloatArray:
		return fmt.Sprintf("mapper.treeToValue(_in.get(%q), double[].class)", p.Name), nil
	case TypeStringArray:
		return fmt.Sprintf("mapper.treeToValue(_in.get(%q), String[].class)", p.Name), nil
	case TypeBoolArray:
		return fmt.Sprintf("mapper.treeToValue(_in.get(%q), boolean[].class)", p.Name), nil
	case TypeIntMatrix:
		return fmt.Sprintf("mapper.treeToValue(_in.get(%q), int[][].class)", p.Name), nil
	case TypeStringMatrix:
		return fmt.Sprintf("mapper.treeToValue(_in.get(%q), String[][].class)", p.Name), nil
	case TypeListNode:
		return fmt.Sprintf("deserializeList(_in.get(%q))", p.Name), nil
	case TypeTreeNode:
		return fmt.Sprintf("deserializeTree(_in.get(%q))", p.Name), nil
	}
	return "", fmt.Errorf("unsupported java parameter type: %q", p.Type)
}
