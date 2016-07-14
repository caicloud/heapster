package caicloudcore

import "k8s.io/heapster/metrics/core"

var (
	MetricSetTypeApp = "app"

	LabelAppName = core.LabelDescriptor{
		Key:         "app_name",
		Description: "The name of the application",
	}
)

const UNLIMIT int64 = -1
