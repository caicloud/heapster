package caicloudprocessors

import (
	"net/url"

	"github.com/golang/glog"
	"k8s.io/heapster/metrics/core"
	"k8s.io/heapster/metrics/processors/kubeprocessors"
)

func GetProcessors(url *url.URL) ([]core.DataProcessor, error) {
	metricsToAggregate := []string{
		core.MetricCpuUsageRate.Name,
		core.MetricMemoryUsage.Name,
		core.MetricCpuRequest.Name,
		core.MetricCpuLimit.Name,
		core.MetricMemoryRequest.Name,
		core.MetricMemoryLimit.Name,
	}

	metricsToAggregateForNode := []string{
		core.MetricCpuRequest.Name,
		core.MetricCpuLimit.Name,
		core.MetricMemoryRequest.Name,
		core.MetricMemoryLimit.Name,
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
	namespaceBasedEnricher, err := NewNamespaceBasedEnricher(url)
	if err != nil {
		glog.Fatalf("Failed to create NamespaceBasedEnricher: %v", err)
		return nil, err
	}

	dataProcessors = append(dataProcessors, podBasedEnricher)
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
			MetricsToAggregate: metricsToAggregate,
		},
		&kubeprocessors.NamespaceAggregator{
			MetricsToAggregate: metricsToAggregate,
		},
		&kubeprocessors.NodeAggregator{
			MetricsToAggregate: metricsToAggregateForNode,
		},
	)

	return dataProcessors, nil
}
