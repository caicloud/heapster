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

var MetricCpuAvailable = core.Metric{
	MetricDescriptor: core.MetricDescriptor{
		Name:        "cpu/available",
		Description: "CPU available in millicores.",
		Type:        core.MetricGauge,
		ValueType:   core.ValueInt64,
		Units:       core.UnitsCount,
	},
}

var MetricMemoryAvailable = core.Metric{
	MetricDescriptor: core.MetricDescriptor{
		Name:        "memory/available",
		Description: "Memory available in bytes. This metric is not equal limit - usage.",
		Type:        core.MetricGauge,
		ValueType:   core.ValueInt64,
		Units:       core.UnitsBytes,
	},
}

var AdditionalMetrics = []core.Metric{
	core.MetricCpuRequest,
	core.MetricCpuLimit,
	MetricCpuAvailable,
	core.MetricMemoryRequest,
	core.MetricMemoryLimit,
	MetricMemoryAvailable,
}

var AllMetrics = append(append(append(append(core.StandardMetrics, AdditionalMetrics...), core.RateMetrics...), core.LabeledMetrics...),
	core.NodeAutoscalingMetrics...)
