package caicloudprocessors

import (
	"fmt"

	"k8s.io/heapster/metrics/core"
)

func aggregateRootMetric(src, dst *core.MetricSet, metricsToAggregate map[string]bool) error {
	const rootKey = "/"
	const logsKey = "logs"
	rootMetric := make(map[string]core.MetricValue)
	logsMetric := make(map[string]core.MetricValue)
	for _, labeledMetric := range dst.LabeledMetrics {
		if metricsToAggregate[labeledMetric.Name] {
			if fsKey, ok := labeledMetric.Labels[core.LabelResourceID.Key]; ok {
				if fsKey == rootKey {
					rootMetric[labeledMetric.Name] = labeledMetric.MetricValue
				} else if fsKey == logsKey {
					logsMetric[labeledMetric.Name] = labeledMetric.MetricValue
				}
			}
		}
	}
	for _, labeledMetric := range src.LabeledMetrics {
		if metricsToAggregate[labeledMetric.Name] {
			// TODO(liubog2008): aggregate volume resource
			if v, ok := labeledMetric.Labels[core.LabelResourceID.Key]; ok {
				var aggregatedMetric map[string]core.MetricValue
				if v == rootKey {
					aggregatedMetric = rootMetric
				} else if v == logsKey {
					aggregatedMetric = logsMetric
				} else {
					continue
				}
				if metric, ok := aggregatedMetric[labeledMetric.Name]; ok {
					if metric.ValueType != labeledMetric.MetricValue.ValueType {
						return fmt.Errorf("Aggregator: type not supported in %s", labeledMetric.Name)
					}
					if metric.ValueType == core.ValueInt64 {
						metric.IntValue += labeledMetric.MetricValue.IntValue
					} else if metric.ValueType == core.ValueFloat {
						metric.FloatValue += labeledMetric.MetricValue.FloatValue
					} else {
						return fmt.Errorf("Aggregator: type not supported in %s", labeledMetric.Name)
					}
					aggregatedMetric[labeledMetric.Name] = metric
				} else {
					aggregatedMetric[labeledMetric.Name] = labeledMetric.MetricValue
				}
			}
		}
	}
	rootLabels := map[string]string{core.LabelResourceID.Key: rootKey}
	logsLabels := map[string]string{core.LabelResourceID.Key: logsKey}
	for name, metricValue := range rootMetric {
		dst.LabeledMetrics = append(dst.LabeledMetrics, core.LabeledMetric{
			Name:        name,
			Labels:      rootLabels,
			MetricValue: metricValue,
		})
	}
	for name, metricValue := range logsMetric {
		dst.LabeledMetrics = append(dst.LabeledMetrics, core.LabeledMetric{
			Name:        name,
			Labels:      logsLabels,
			MetricValue: metricValue,
		})
	}
	return nil
}

var labeledMetricsName = map[string]bool{
	core.MetricFilesystemUsage.Name:     true,
	core.MetricFilesystemLimit.Name:     true,
	core.MetricFilesystemAvailable.Name: true,
}

func Aggregate(src, dst *core.MetricSet, metricsToAggregate []string) error {
	for _, metricName := range metricsToAggregate {
		metricValue, found := src.MetricValues[metricName]
		if !found {
			continue
		}
		aggregatedValue, found := dst.MetricValues[metricName]
		if found {
			if aggregatedValue.ValueType != metricValue.ValueType {
				return fmt.Errorf("Aggregator: type not supported in %s", metricName)
			}

			if aggregatedValue.ValueType == core.ValueInt64 {
				aggregatedValue.IntValue += metricValue.IntValue
			} else if aggregatedValue.ValueType == core.ValueFloat {
				aggregatedValue.FloatValue += metricValue.FloatValue
			} else {
				return fmt.Errorf("Aggregator: type not supported in %s", metricName)
			}
		} else {
			aggregatedValue = metricValue
		}
		dst.MetricValues[metricName] = aggregatedValue
	}
	labeledMetricNameExist := map[string]bool{}
	for _, metricName := range metricsToAggregate {
		if labeledMetricsName[metricName] {
			labeledMetricNameExist[metricName] = true
		}
	}
	return aggregateRootMetric(src, dst, labeledMetricNameExist)
}

func intValue(value int64) core.MetricValue {
	return core.MetricValue{
		IntValue:   value,
		MetricType: core.MetricGauge,
		ValueType:  core.ValueInt64,
	}
}

// a < b,  return true
// a >= b, return false
func less(a *core.MetricValue, b *core.MetricValue) bool {
	if a.ValueType == core.ValueInt64 {
		if b.ValueType == core.ValueInt64 {
			return a.IntValue < b.IntValue
		} else {
			return float64(a.IntValue) < float64(b.FloatValue)
		}
	} else {
		if b.ValueType == core.ValueInt64 {
			return float64(a.FloatValue) < float64(b.IntValue)
		} else {
			return a.FloatValue < b.FloatValue
		}
	}
}
