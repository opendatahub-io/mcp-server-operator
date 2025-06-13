package gvk

import (
	"k8s.io/apimachinery/pkg/runtime/schema"
)

var (
	MCPServer = schema.GroupVersionKind{
		Group:   "mcpserver.opendatahub.io",
		Kind:    "MCPServer",
		Version: "v1",
	}
)
