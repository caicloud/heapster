package caicloudprocessors

import (
	"net/url"

	"github.com/golang/glog"
	"k8s.io/heapster/metrics/core"
	"k8s.io/heapster/metrics/core/caicloud"
	"k8s.io/heapster/metrics/processors/kubeprocessors"
)

func GetProcessors(url *url.URL) ([]core.DataProcessor, error) {
	nsAndAppMetricsToAggregate := []string{
		core.MetricCpuUsageRate.Name,

		core.MetricMemoryUsage.Name,
		core.MetricMemoryWorkingSet.Name,
		core.MetricMemoryPageFaultsRate.Name,
		core.MetricMemoryMajorPageFaultsRate.Name,

		core.MetricFilesystemUsage.Name,

		core.MetricNetworkRxRate.Name,
		core.MetricNetworkTxRate.Name,
		core.MetricNetworkRxErrorsRate.Name,
		core.MetricNetworkTxErrorsRate.Name,
	}
	clusterMetricsToAggregate := []string{
		core.MetricCpuUsageRate.Name,
		core.MetricCpuLimit.Name,
		caicloudcore.MetricCpuAvailable.Name,

		core.MetricMemoryUsage.Name,
		core.MetricMemoryLimit.Name,
		caicloudcore.MetricMemoryAvailable.Name,

		core.MetricFilesystemUsage.Name,
		core.MetricFilesystemLimit.Name,
		core.MetricFilesystemAvailable.Name,
	}

	dataProcessors := []core.DataProcessor{
		// Convert cumulaties to rate
		kubeprocessors.NewRateCalculator(core.RateMetricsMapping),
	}

	podBasedEnricher, err := NewPodBasedEnricher(url)
	if err != nil {
		glog.Fatalf("Failed to create PodBasedEnricher: %v", err)
		return nil, err
	}
	dataProcessors = append(dataProcessors, podBasedEnricher)

	namespaceBasedEnricher, err := NewNamespaceBasedEnricher(url)
	if err != nil {
		glog.Fatalf("Failed to create NamespaceBasedEnricher: %v", err)
		return nil, err
	}
	dataProcessors = append(dataProcessors, namespaceBasedEnricher)

	dataProcessors = append(dataProcessors,
		&PodAggregator{
			MetricsToAggregate: []string{
				core.MetricCpuUsageRate.Name,

				core.MetricMemoryUsage.Name,
				core.MetricMemoryWorkingSet.Name,
				core.MetricMemoryPageFaultsRate.Name,
				core.MetricMemoryMajorPageFaultsRate.Name,

				core.MetricFilesystemUsage.Name,
				core.MetricFilesystemAvailable.Name,

				core.MetricNetworkRxRate.Name,
				core.MetricNetworkTxRate.Name,
				core.MetricNetworkRxErrorsRate.Name,
				core.MetricNetworkTxErrorsRate.Name,

				core.MetricCpuRequest.Name,
				core.MetricCpuLimit.Name,
				core.MetricMemoryRequest.Name,
				core.MetricMemoryLimit.Name,
			},
		},
		&ApplicationAggregator{
			MetricsToAggregate: nsAndAppMetricsToAggregate,
		},
		&kubeprocessors.NamespaceAggregator{
			MetricsToAggregate: nsAndAppMetricsToAggregate,
		},
		&ClusterAggregator{
			MetricsToAggregate: clusterMetricsToAggregate,
		},
	)

	nodeAutoscalingEnricher, err := kubeprocessors.NewNodeAutoscalingEnricher(url)
	if err != nil {
		glog.Fatalf("Failed to create NodeAutoscalingEnricher: %v", err)
		return nil, err
	}
	dataProcessors = append(dataProcessors, nodeAutoscalingEnricher)

	return dataProcessors, nil
}
