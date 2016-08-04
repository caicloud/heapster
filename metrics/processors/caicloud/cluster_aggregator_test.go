package caicloudprocessors

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"k8s.io/heapster/metrics/core"
)

func TestClusterAggregate(t *testing.T) {
	batch := core.DataBatch{
		Timestamp: time.Now(),
		MetricSets: map[string]*core.MetricSet{
			core.NodeKey("node1"): {
				Labels: map[string]string{
					core.LabelMetricSetType.Key: core.MetricSetTypeNamespace,
					core.LabelNamespaceName.Key: "ns1",
				},
				MetricValues: map[string]core.MetricValue{
					"m1": {
						ValueType:  core.ValueInt64,
						MetricType: core.MetricGauge,
						IntValue:   10,
					},
					"m2": {
						ValueType:  core.ValueInt64,
						MetricType: core.MetricGauge,
						IntValue:   222,
					},
				},
				LabeledMetrics: []core.LabeledMetric{
					core.LabeledMetric{
						Name: core.MetricFilesystemLimit.Name,
						Labels: map[string]string{
							"resource_id": "/",
						},
						MetricValue: core.MetricValue{
							ValueType:  core.ValueInt64,
							MetricType: core.MetricGauge,
							IntValue:   222,
						},
					},
					core.LabeledMetric{
						Name: core.MetricFilesystemAvailable.Name,
						Labels: map[string]string{
							"resource_id": "/",
						},
						MetricValue: core.MetricValue{
							ValueType:  core.ValueInt64,
							MetricType: core.MetricGauge,
							IntValue:   222,
						},
					},
				},
			},

			core.NodeKey("node2"): {
				Labels: map[string]string{
					core.LabelMetricSetType.Key: core.MetricSetTypeNamespace,
					core.LabelNamespaceName.Key: "ns1",
				},
				MetricValues: map[string]core.MetricValue{
					"m1": {
						ValueType:  core.ValueInt64,
						MetricType: core.MetricGauge,
						IntValue:   100,
					},
					"m3": {
						ValueType:  core.ValueInt64,
						MetricType: core.MetricGauge,
						IntValue:   30,
					},
				},
				LabeledMetrics: []core.LabeledMetric{
					core.LabeledMetric{
						Name: core.MetricFilesystemAvailable.Name,
						Labels: map[string]string{
							"resource_id": "/",
						},
						MetricValue: core.MetricValue{
							ValueType:  core.ValueInt64,
							MetricType: core.MetricGauge,
							IntValue:   100,
						},
					},
					core.LabeledMetric{
						Name: core.MetricFilesystemLimit.Name,
						Labels: map[string]string{
							"resource_id": "/",
						},
						MetricValue: core.MetricValue{
							ValueType:  core.ValueInt64,
							MetricType: core.MetricGauge,
							IntValue:   22,
						},
					},
				},
			},
		},
	}
	processor := ClusterAggregator{
		MetricsToAggregate: []string{core.MetricFilesystemLimit.Name, core.MetricFilesystemAvailable.Name},
	}
	result, err := processor.Process(&batch)
	assert.NoError(t, err)
	cluster, found := result.MetricSets[core.ClusterKey()]
	assert.True(t, found)

	fmt.Println(cluster.LabeledMetrics)
}
