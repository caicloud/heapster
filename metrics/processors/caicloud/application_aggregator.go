package caicloudprocessors

import (
	"github.com/golang/glog"
	"k8s.io/heapster/metrics/core"
	"k8s.io/heapster/metrics/core/caicloud"
)

type ApplicationAggregator struct {
	MetricsToAggregate []string
}

func (a *ApplicationAggregator) Name() string {
	return "application_aggregator"
}

func (a *ApplicationAggregator) Process(batch *core.DataBatch) (*core.DataBatch, error) {
	apps := make(map[string]*core.MetricSet)
	for key, metricSet := range batch.MetricSets {
		if metricSetType, found := metricSet.Labels[core.LabelMetricSetType.Key]; found && metricSetType == core.MetricSetTypePod {
			// Aggregating pods
			appName, found := metricSet.Labels[caicloudcore.LabelAppName.Key]
			namespaceName, found2 := metricSet.Labels[core.LabelNamespaceName.Key]
			if found && found2 {
				appKey := caicloudcore.AppKey(namespaceName, appName)
				app, found := apps[appKey]
				if !found {
					if appFromBatch, found := batch.MetricSets[appKey]; found {
						app = appFromBatch
					} else {
						app = appMetricSet(namespaceName, appName)
						apps[appKey] = app
					}
				}
				if err := Aggregate(metricSet, app, a.MetricsToAggregate); err != nil {
					return nil, err
				}
			} else {
				glog.Errorf("No namespace and/or app info in pod %s: %v", key, metricSet.Labels)
			}
		}
	}
	for key, val := range apps {
		batch.MetricSets[key] = val
	}
	return batch, nil
}

func appMetricSet(namespaceName, appName string) *core.MetricSet {
	return &core.MetricSet{
		MetricValues: make(map[string]core.MetricValue),
		Labels: map[string]string{
			core.LabelMetricSetType.Key:   caicloudcore.MetricSetTypeApp,
			core.LabelNamespaceName.Key:   namespaceName,
			caicloudcore.LabelAppName.Key: appName,
		},
	}
}
