package tools

// IsMutatingCall returns true if the given tool call would modify the graph.
// toolName is the MCP tool name, args is the arguments map from the request.
func IsMutatingCall(toolName string, args map[string]any) bool {
	switch toolName {
	case "triple":
		a, _ := args["action"].(string)
		return a == "add" || a == "remove"
	case "sparql":
		q, _ := args["query"].(string)
		return len(q) > 0 && isUpdate(q)
	case "graph":
		a, _ := args["action"].(string)
		return a == "create" || a == "delete" || a == "clear" || a == "load" || a == "import" || a == "migrate"
	}
	return false
}
