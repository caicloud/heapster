package caicloudprocessors

import "k8s.io/heapster/metrics/core"

type ClusterAggregator struct {
	MetricsToAggregate []string
}

func (this *ClusterAggregator) Name() string {
	return "cluster_aggregator"
}

func (this *ClusterAggregator) Process(batch *core.DataBatch) (*core.DataBatch, error) {
	clusterKey := core.ClusterKey()
	cluster := clusterMetricSet()
	for _, metricSet := range batch.MetricSets {
		if metricSetType, found := metricSet.Labels[core.LabelMetricSetType.Key]; found &&
			metricSetType == core.MetricSetTypeNode {
			if err := Aggregate(metricSet, cluster, this.MetricsToAggregate); err != nil {
				return nil, err
			}
		}
	}
	batch.MetricSets[clusterKey] = cluster
	return batch, nil
}

func clusterMetricSet() *core.MetricSet {
	return &core.MetricSet{
		MetricValues: make(map[string]core.MetricValue),
		Labels: map[string]string{
			core.LabelMetricSetType.Key: core.MetricSetTypeCluster,
		},
	}
}
