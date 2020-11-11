/*


Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package setup

import (
	"context"
	"flag"
	"fmt"
	eventloggerv1 "github.com/bakito/k8s-event-logger-operator/api/v1"
	cnst "github.com/bakito/k8s-event-logger-operator/pkg/constants"
	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

func (r *Reconciler) createOrReplacePod(ctx context.Context, cr *eventloggerv1.EventLogger, pod *corev1.Pod,
	reqLogger logr.Logger) (bool, error) {

	podList := &corev1.PodList{}
	opts := []client.ListOption{
		client.InNamespace(cr.Namespace),
		client.MatchingLabels(map[string]string{
			"app":        loggerName(cr),
			"created-by": "eventlogger"}),
	}
	err := r.List(ctx, podList, opts...)

	if err != nil {
		return false, err
	}

	replacePod := false
	if len(podList.Items) == 1 {
		op := podList.Items[0]
		replacePod = podChanged(&op, pod)
	}

	if replacePod || len(podList.Items) > 1 {
		for _, p := range podList.Items {
			reqLogger.Info(fmt.Sprintf("Deleting %s", pod.Kind), "namespace", pod.GetNamespace(), "name", pod.GetName())
			err = r.Delete(ctx, &p, &client.DeleteOptions{GracePeriodSeconds: &gracePeriod})
			if err != nil {
				return false, err
			}
		}
		podList = &corev1.PodList{}
	}

	if len(podList.Items) == 0 {
		// Set EventLogger cr as the owner and controller
		if err := controllerutil.SetControllerReference(cr, pod, r.Scheme); err != nil {
			return false, err
		}
		reqLogger.Info(fmt.Sprintf("Creating a new %s", pod.Kind), "namespace", pod.GetNamespace(), "name", pod.GetName())
		err = r.Create(ctx, pod)
		if err != nil {
			return false, err
		}
		return true, nil
	}

	return false, nil
}

// podForCR returns a pod with the same name/namespace as the cr
func (r *Reconciler) podForCR(cr *eventloggerv1.EventLogger) *corev1.Pod {
	metricsAddrFlag := flag.Lookup(cnst.ArgMetricsAddr)
	var metricsAddr string
	if metricsAddrFlag != nil {
		metricsAddr = metricsAddrFlag.Value.String()

	}
	if metricsAddr == "" {
		metricsAddr = cnst.DefaultMetricsAddr
	}
	metricsPort := metricsAddr[:1]

	labels := make(map[string]string)
	for k, v := range cr.Spec.Labels {
		labels[k] = v
	}
	labels["app"] = loggerName(cr)
	labels["created-by"] = "eventlogger"

	annotations := make(map[string]string)
	for k, v := range cr.Spec.Annotations {
		annotations[k] = v
	}
	if cr.Spec.ScrapeMetrics != nil && *cr.Spec.ScrapeMetrics {
		annotations["prometheus.io/port"] = metricsPort
		annotations["prometheus.io/scrape"] = "true"
	}

	watchNamespace := cr.GetNamespace()
	if cr.Spec.Namespace != nil {
		watchNamespace = *cr.Spec.Namespace
	}

	saccName := loggerName(cr)
	if cr.Spec.ServiceAccount != "" {
		saccName = cr.Spec.ServiceAccount
	}

	pod := &corev1.Pod{
		TypeMeta: metav1.TypeMeta{
			Kind: "Pod",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:        loggerName(cr) + "-" + randString(),
			Namespace:   cr.Namespace,
			Labels:      labels,
			Annotations: annotations,
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Name:            "event-logger",
					Image:           r.eventLoggerImage,
					ImagePullPolicy: corev1.PullAlways,
					Command:         []string{"/opt/go/k8s-event-logger"},
					Args: []string{
						"--" + cnst.ArgConfigName, cr.Name,
						"--" + cnst.ArgMetricsAddr, metricsAddr,
						"--" + cnst.ArgEnableLoggerMode, "true",
					},
					Env: []corev1.EnvVar{
						{Name: cnst.EnvWatchNamespace, Value: watchNamespace},
					},
					Resources: corev1.ResourceRequirements{
						Requests: corev1.ResourceList{corev1.ResourceCPU: r.podReqCPU, corev1.ResourceMemory: r.podReqMem},
						Limits:   corev1.ResourceList{corev1.ResourceCPU: r.podMaxCPU, corev1.ResourceMemory: r.podMaxMem},
					},
				},
			},
			ServiceAccountName: saccName,
			NodeSelector:       cr.Spec.NodeSelector,
		},
	}
	return pod
}
