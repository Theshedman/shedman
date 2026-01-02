package resolver

// TopologicalSort returns packages in dependency order (dependencies first)
func TopologicalSort(nodes map[string]*DependencyNode) []string {
	var result []string
	visited := make(map[string]bool)
	visiting := make(map[string]bool) // For cycle detection

	var visit func(name string)
	visit = func(name string) {
		if visited[name] {
			return
		}
		if visiting[name] {
			// Cycle detected - skip to avoid infinite loop
			return
		}

		visiting[name] = true

		// Visit dependencies first
		if node, ok := nodes[name]; ok {
			for _, dep := range node.Children {
				visit(dep)
			}
		}

		visiting[name] = false
		visited[name] = true
		result = append(result, name)
	}

	// Visit all nodes
	for name := range nodes {
		visit(name)
	}

	return result
}