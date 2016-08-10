package caicloudprocessors

import (
	"github.com/golang/glog"
	"k8s.io/heapster/metrics/core"
	"k8s.io/heapster/metrics/core/caicloud"
)

type PodAggregator struct {
	MetricsToAggregate []string
}

func (p *PodAggregator) Name() string {
	return "pod_aggregator"
}

func (p *PodAggregator) Process(batch *core.DataBatch) (*core.DataBatch, error) {
	pods := make(map[string]*core.MetricSet)

	for key, metricSet := range batch.MetricSets {

		if metricSetType, found := metricSet.Labels[core.LabelMetricSetType.Key]; found && metricSetType == core.MetricSetTypePodContainer {
			podName, found := metricSet.Labels[core.LabelPodName.Key]
			_, found2 := metricSet.Labels[caicloudcore.LabelAppName.Key]
			namespaceName, found3 := metricSet.Labels[core.LabelNamespaceName.Key]
			if found && found2 && found3 {
				podKey := core.PodKey(namespaceName, podName)
				pod, found := pods[podKey]
				if !found {
					if podFromBatch, found := batch.MetricSets[podKey]; found {
						pod = podFromBatch
					} else {
						pod = podMetricSet(metricSet.Labels)
						pods[podKey] = pod
					}
					if err := Aggregate(metricSet, pod, p.MetricsToAggregate); err != nil {
						return nil, err
					}
				}
			} else {
				glog.Errorf("No namespace and/or app and/or pod info in pod %s: %v", key, metricSet.Labels)
			}
		}
	}
	for key, val := range pods {
		batch.MetricSets[key] = val
	}
	return batch, nil
}

var labelsToPopulate = []core.LabelDescriptor{
	core.LabelNamespaceName,
	core.LabelPodNamespace,
	caicloudcore.LabelAppName,
	core.LabelPodName,
	core.LabelNodename,
	core.LabelHostname,
	core.LabelHostID,
}

func podMetricSet(labels map[string]string) *core.MetricSet {
	newLabels := map[string]string{
		core.LabelMetricSetType.Key: core.MetricSetTypePod,
	}
	for _, l := range labelsToPopulate {
		if val, ok := labels[l.Key]; ok {
			newLabels[l.Key] = val
		}
	}
	return &core.MetricSet{
		MetricValues: make(map[string]core.MetricValue),
		Labels:       newLabels,
	}
}
