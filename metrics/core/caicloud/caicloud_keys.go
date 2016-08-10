package caicloudcore

import (
	"fmt"
)

// MetricsSet keys inside of DataBatch. The structure of the returned string is
// an implementation detail and no component should rely on it as it may change
// anytime. It it only guaranteed that it is unique for the unique combination of
// passed parameters.

func AppKey(namespaceName, appName string) string {
	return fmt.Sprintf("namespace:%s/app:%s", namespaceName, appName)
}

func NamespaceKey(namespaceName string) string {
	return fmt.Sprintf("namespace:%s", namespaceName)
}

func NodeKey(node string) string {
	return fmt.Sprintf("node:%s", node)
}

func NodeContainerKey(node, container string) string {
	return fmt.Sprintf("node:%s/container:%s", node, container)
}

func ClusterKey() string {
	return "cluster"
}
