package caicloudprocessors

import (
	"net/url"
	"time"

	"github.com/golang/glog"
	kubeconfig "k8s.io/heapster/common/kubernetes"
	"k8s.io/heapster/metrics/core"
	"k8s.io/heapster/metrics/core/caicloud"
	kubeapi "k8s.io/kubernetes/pkg/api"
	"k8s.io/kubernetes/pkg/api/resource"
	"k8s.io/kubernetes/pkg/client/cache"
	kubeclient "k8s.io/kubernetes/pkg/client/unversioned"
	"k8s.io/kubernetes/pkg/fields"
)

type NamespaceBasedEnricher struct {
	nsStore     cache.Store
	nsReflector *cache.Reflector

	quotaStore     cache.Store
	quotaReflector *cache.Reflector
}

func (n *NamespaceBasedEnricher) Name() string {
	return "namespace_based_enricher"
}

func (n *NamespaceBasedEnricher) Process(batch *core.DataBatch) (*core.DataBatch, error) {
	rl := n.getNamespaceRequestAndLimit()
	for _, ms := range batch.MetricSets {
		n.addNamespaceInfo(ms, rl)
	}
	return batch, nil
}

var requestAndLimit = map[kubeapi.ResourceName]string{
	kubeapi.ResourceLimitsCPU:      core.MetricCpuLimit.Name,
	kubeapi.ResourceLimitsMemory:   core.MetricMemoryLimit.Name,
	kubeapi.ResourceRequestsCPU:    core.MetricCpuRequest.Name,
	kubeapi.ResourceRequestsMemory: core.MetricMemoryRequest.Name,
}

// Adds UID to all namespaced elements.
func (n *NamespaceBasedEnricher) addNamespaceInfo(metricSet *core.MetricSet, rl map[string]kubeapi.ResourceList) {
	if metricSetType, found := metricSet.Labels[core.LabelMetricSetType.Key]; found &&
		(metricSetType == core.MetricSetTypePodContainer ||
			metricSetType == core.MetricSetTypePod ||
			metricSetType == core.MetricSetTypeNamespace) {

		if namespaceName, found := metricSet.Labels[core.LabelNamespaceName.Key]; found {
			nsObj, exists, err := n.nsStore.GetByKey(namespaceName)
			if exists && err == nil {
				namespace, ok := nsObj.(*kubeapi.Namespace)
				if ok {
					metricSet.Labels[core.LabelPodNamespaceUID.Key] = string(namespace.UID)
					if metricSetType == core.MetricSetTypeNamespace {
						if resourceList, found := rl[namespaceName]; found {
							for key, metirc := range requestAndLimit {
								if val, found := resourceList[key]; found {
									var v int64
									switch key {
									case kubeapi.ResourceRequestsCPU:
										fallthrough
									case kubeapi.ResourceLimitsCPU:
										v = val.MilliValue()
									case kubeapi.ResourceRequestsMemory:
										fallthrough
									case kubeapi.ResourceLimitsMemory:
										v = val.Value()
									}
									glog.V(3).Infof("add namespace %v quota info, %v: %v", namespaceName, metirc, v)
									metricSet.MetricValues[metirc] = intValue(v)
								} else {
									metricSet.MetricValues[metirc] = intValue(caicloudcore.UNLIMIT)
								}
							}
						} else {
							for _, name := range requestAndLimit {
								metricSet.MetricValues[name] = intValue(caicloudcore.UNLIMIT)
							}
						}
					}
				} else {
					glog.Errorf("Wrong namespace store content")
				}
			} else {
				if err != nil {
					glog.Warningf("Failed to get namespace %s: %v", namespaceName, err)
				} else if !exists {
					glog.Warningf("Namespace doesn't exist: %s", namespaceName)
				}
			}
		}
	}
}

func (n *NamespaceBasedEnricher) getNamespaceRequestAndLimit() map[string]kubeapi.ResourceList {
	rl := make(map[string]kubeapi.ResourceList)
	quotas := n.quotaStore.List()
	for _, quotaObj := range quotas {
		if quota, ok := quotaObj.(*kubeapi.ResourceQuota); ok {
			m := minResource(quota.Spec.Hard, rl[quota.Namespace])
			glog.V(3).Infof("namespace: %v, quota hard: %v, request and limt: %v", quota.Namespace, quota.Spec.Hard, rl[quota.Namespace])
			rl[quota.Namespace] = m
		} else {
			glog.Errorf("Wrong quota store content")
		}
	}
	return rl
}

var resourceToPopulate = []kubeapi.ResourceName{
	kubeapi.ResourceCPU,
	kubeapi.ResourceMemory,
	kubeapi.ResourceLimitsCPU,
	kubeapi.ResourceLimitsMemory,
	kubeapi.ResourceRequestsCPU,
	kubeapi.ResourceRequestsMemory,
}

func minResource(a kubeapi.ResourceList, b kubeapi.ResourceList) kubeapi.ResourceList {
	result := kubeapi.ResourceList{}
	for _, key := range resourceToPopulate {
		var name kubeapi.ResourceName
		switch key {
		case kubeapi.ResourceCPU:
			name = kubeapi.ResourceRequestsCPU
		case kubeapi.ResourceMemory:
			name = kubeapi.ResourceRequestsMemory
		case kubeapi.ResourceLimitsCPU:
			fallthrough
		case kubeapi.ResourceLimitsMemory:
			fallthrough
		case kubeapi.ResourceRequestsCPU:
			fallthrough
		case kubeapi.ResourceRequestsMemory:
			name = key
		}
		var (
			aValue   *resource.Quantity
			bValue   *resource.Quantity
			resValue *resource.Quantity
		)
		if val, found := a[key]; found {
			aValue = &val
		} else {
			aValue = nil
		}
		if val, found := b[key]; found {
			bValue = &val
		} else {
			bValue = nil
		}
		if val, found := result[key]; found {
			resValue = &val
		} else {
			resValue = nil
		}

		val := min(aValue, bValue, resValue)
		if val != nil {
			result[name] = *(val.Copy())
		}
	}
	return result
}

func min(vals ...*resource.Quantity) *resource.Quantity {
	if len(vals) == 0 {
		return nil
	} else if len(vals) == 1 {
		return vals[0]
	} else {
		var min *resource.Quantity = nil
		for _, val := range vals {
			if min == nil {
				min = val
			}
			if val != nil && min.Cmp(*val) > 0 {
				min = val
			}
		}
		return min
	}
}

func NewNamespaceBasedEnricher(url *url.URL) (*NamespaceBasedEnricher, error) {
	kubeConfig, err := kubeconfig.GetKubeClientConfig(url)
	if err != nil {
		return nil, err
	}
	kubeClient := kubeclient.NewOrDie(kubeConfig)

	nsLw := cache.NewListWatchFromClient(kubeClient, "namespaces", kubeapi.NamespaceAll, fields.Everything())
	nsStore := cache.NewStore(cache.MetaNamespaceKeyFunc)
	nsReflector := cache.NewReflector(nsLw, &kubeapi.Namespace{}, nsStore, time.Hour)
	nsReflector.Run()

	quotaLw := cache.NewListWatchFromClient(kubeClient, "resourcequotas", kubeapi.NamespaceAll, fields.Everything())
	quotaStore := cache.NewStore(cache.MetaNamespaceKeyFunc)
	quotaReflector := cache.NewReflector(quotaLw, &kubeapi.ResourceQuota{}, quotaStore, time.Hour)
	quotaReflector.Run()

	return &NamespaceBasedEnricher{
		nsStore:     nsStore,
		nsReflector: nsReflector,

		quotaStore:     quotaStore,
		quotaReflector: quotaReflector,
	}, nil
}
