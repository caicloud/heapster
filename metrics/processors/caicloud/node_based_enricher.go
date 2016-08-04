package caicloudprocessors

import (
	"net/url"
	"time"

	"github.com/golang/glog"
	kubeconfig "k8s.io/heapster/common/kubernetes"
	"k8s.io/heapster/metrics/core"
	"k8s.io/heapster/metrics/core/caicloud"
	kubeapi "k8s.io/kubernetes/pkg/api"
	"k8s.io/kubernetes/pkg/client/cache"
	kubeclient "k8s.io/kubernetes/pkg/client/unversioned"
	"k8s.io/kubernetes/pkg/fields"
)

type NodeBasedEnricher struct {
	nodeLister *cache.StoreToNodeLister
}

func (n *NodeBasedEnricher) Name() string {
	return "node_based_enricher"
}

func (n *NodeBasedEnricher) Process(batch *core.DataBatch) (*core.DataBatch, error) {
	nodes := make(map[string]*core.MetricSet, len(batch.MetricSets))
	nodesMetas, err := n.nodeLister.List()
	if err != nil {
		glog.Errorf("error while listing nodes: %v", err)
		return batch, err
	}
	unschedulable := make(map[string]bool)
	for _, node := range nodesMetas.Items {
		unschedulable[node.Name] = node.Spec.Unschedulable
	}

	for _, metricSet := range batch.MetricSets {
		if metricSetType, found := metricSet.Labels[core.LabelMetricSetType.Key]; found && metricSetType == core.MetricSetTypeNode {
			nodeName := metricSet.Labels[core.LabelNodename.Key]
			if unschedulable[nodeName] {
				metricSet.MetricValues[caicloudcore.MetricCpuAvailable.Name] = intValue(0)
				metricSet.MetricValues[caicloudcore.MetricMemoryAvailable.Name] = intValue(0)
				metricSet.LabeledMetrics = append(metricSet.LabeledMetrics,
					core.LabeledMetric{
						Name:   core.MetricFilesystemAvailable.Name,
						Labels: map[string]string{core.LabelResourceID.Key: "/"},
						MetricValue: core.MetricValue{
							ValueType:  core.ValueInt64,
							MetricType: core.MetricFilesystemAvailable.Type,
							IntValue:   0,
						},
					})
			} else {
				cpuUsageRate := metricSet.MetricValues[core.MetricCpuUsageRate.Name]
				cpuLimit := metricSet.MetricValues[core.MetricCpuLimit.Name]
				metricSet.MetricValues[caicloudcore.MetricCpuAvailable.Name] = intValue(cpuLimit.IntValue - cpuUsageRate.IntValue)
				memoryUsage := metricSet.MetricValues[core.MetricMemoryUsage.Name]
				memoryLimit := metricSet.MetricValues[core.MetricMemoryLimit.Name]
				metricSet.MetricValues[caicloudcore.MetricMemoryAvailable.Name] = intValue(memoryLimit.IntValue - memoryUsage.IntValue)
			}
		}
	}
	for k, v := range nodes {
		batch.MetricSets[k] = v
	}
	return batch, nil
}

func NewNodeBasedEnricher(url *url.URL) (*NodeBasedEnricher, error) {
	kubeConfig, err := kubeconfig.GetKubeClientConfig(url)
	if err != nil {
		return nil, err
	}
	kubeClient := kubeclient.NewOrDie(kubeConfig)

	lw := cache.NewListWatchFromClient(kubeClient, "nodes", kubeapi.NamespaceAll, fields.Everything())
	nodeLister := &cache.StoreToNodeLister{Store: cache.NewStore(cache.MetaNamespaceKeyFunc)}
	reflector := cache.NewReflector(lw, &kubeapi.Node{}, nodeLister.Store, time.Hour)
	reflector.Run()

	return &NodeBasedEnricher{
		nodeLister: nodeLister,
	}, nil
}
