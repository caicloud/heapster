package caicloudsource

import (
	"fmt"
	"net/url"
	"time"

	"github.com/golang/glog"
	. "k8s.io/heapster/metrics/core"
	"k8s.io/heapster/metrics/sources/kubelet"
	"k8s.io/heapster/metrics/sources/summary"
	kubeapi "k8s.io/kubernetes/pkg/api"
	"k8s.io/kubernetes/pkg/client/cache"
	kubeclient "k8s.io/kubernetes/pkg/client/unversioned"
	"k8s.io/kubernetes/pkg/fields"
)

type caicloudMetricsSource struct {
	node          summary.NodeInfo
	summary       MetricsSource
	kubeletClient *kubelet.KubeletClient
}

func NewCaicloudMetricsSource(node summary.NodeInfo, client *kubelet.KubeletClient, summary MetricsSource) MetricsSource {
	return &caicloudMetricsSource{
		node:          node,
		summary:       summary,
		kubeletClient: client,
	}
}

func (s *caicloudMetricsSource) Name() string {
	return "caicloud_source"
}

func (s *caicloudMetricsSource) ScrapeMetrics(start, end time.Time) *DataBatch {
	dataBatch := s.summary.ScrapeMetrics(start, end)
	machineInfo, err := s.kubeletClient.GetMachineInfo(s.node.Host)
	if err != nil {
		glog.Errorf("can't get machine info from kubelet")
		return dataBatch
	}
	key := NodeKey(s.node.NodeName)
	if metricSet, found := dataBatch.MetricSets[key]; found {
		var cpuLimit int64 = int64(machineInfo.NumCores) * 1000

		s.addIntMetric(metricSet, &MetricCpuLimit, cpuLimit)
		s.addIntMetric(metricSet, &MetricMemoryLimit, machineInfo.MemoryCapacity)
	}
	return dataBatch
}

func (s *caicloudMetricsSource) addIntMetric(metrics *MetricSet, metric *Metric, value int64) {
	val := MetricValue{
		ValueType:  ValueInt64,
		MetricType: metric.Type,
		IntValue:   int64(value),
	}
	metrics.MetricValues[metric.Name] = val
}

type caicloudProvider struct {
	nodeLister    *cache.StoreToNodeLister
	reflector     *cache.Reflector
	kubeletClient *kubelet.KubeletClient
}

func (p *caicloudProvider) getNodeInfo(node *kubeapi.Node) (summary.NodeInfo, error) {
	for _, c := range node.Status.Conditions {
		if c.Type == kubeapi.NodeReady && c.Status != kubeapi.ConditionTrue {
			return summary.NodeInfo{}, fmt.Errorf("Node %v is not ready", node.Name)
		}
	}
	info := summary.NodeInfo{
		NodeName: node.Name,
		HostName: node.Name,
		HostID:   node.Spec.ExternalID,
		Host: kubelet.Host{
			Port: p.kubeletClient.GetPort(),
		},
		KubeletVersion: node.Status.NodeInfo.KubeletVersion,
	}

	for _, addr := range node.Status.Addresses {
		if addr.Type == kubeapi.NodeHostName && addr.Address != "" {
			info.HostName = addr.Address
		}
		if addr.Type == kubeapi.NodeInternalIP && addr.Address != "" {
			info.IP = addr.Address
		}
		if addr.Type == kubeapi.NodeLegacyHostIP && addr.Address != "" && info.IP == "" {
			info.IP = addr.Address
		}
	}

	if info.IP == "" {
		return info, fmt.Errorf("Node %v has no valid hostname and/or IP address: %v %v", node.Name, info.HostName, info.IP)
	}

	return info, nil
}

func (p *caicloudProvider) GetMetricsSources() []MetricsSource {
	sources := []MetricsSource{}
	nodes, err := p.nodeLister.List()
	if err != nil {
		glog.Errorf("error while listing nodes: %v", err)
		return sources
	}
	if len(nodes.Items) == 0 {
		glog.Error("No nodes received from APIserver.")
		return sources
	}

	for _, node := range nodes.Items {
		info, err := p.getNodeInfo(&node)
		if err != nil {
			glog.Errorf("%v", err)
			continue
		}
		fallback := kubelet.NewKubeletMetricsSource(
			info.Host,
			p.kubeletClient,
			info.NodeName,
			info.HostName,
			info.HostID,
		)
		summary := summary.NewSummaryMetricsSource(info, p.kubeletClient, fallback)
		sources = append(sources, NewCaicloudMetricsSource(info, p.kubeletClient, summary))
	}
	return sources
}

func NewCaicloudProvider(uri *url.URL) (MetricsSourceProvider, error) {
	// create clients
	kubeConfig, kubeletConfig, err := kubelet.GetKubeConfigs(uri)
	if err != nil {
		return nil, err
	}
	kubeClient := kubeclient.NewOrDie(kubeConfig)
	kubeletClient, err := kubelet.NewKubeletClient(kubeletConfig)
	if err != nil {
		return nil, err
	}
	// watch nodes
	lw := cache.NewListWatchFromClient(kubeClient, "nodes", kubeapi.NamespaceAll, fields.Everything())
	nodeLister := &cache.StoreToNodeLister{Store: cache.NewStore(cache.MetaNamespaceKeyFunc)}
	reflector := cache.NewReflector(lw, &kubeapi.Node{}, nodeLister.Store, time.Hour)
	reflector.Run()

	return &caicloudProvider{
		nodeLister:    nodeLister,
		reflector:     reflector,
		kubeletClient: kubeletClient,
	}, nil
}
