/*
Copyright 2020 The Crossplane Authors.

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

package containerizedworkload

import (
	"context"
	"strings"
	"time"

	"github.com/pkg/errors"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/intstr"
	ctrl "sigs.k8s.io/controller-runtime"

	"github.com/crossplane/crossplane-runtime/pkg/event"
	"github.com/crossplane/crossplane-runtime/pkg/logging"
	oamv1alpha2 "github.com/crossplane/crossplane/apis/oam/v1alpha2"
	"github.com/crossplane/crossplane/apis/workload/v1alpha1"

	"github.com/hasheddan/stack-kubernetes-remote/pkg/reconciler/workload"
)

const (
	reconcileTimeout = 1 * time.Minute
	shortWait        = 30 * time.Second
	longWait         = 1 * time.Minute
)

// Reconcile error strings.
const (
	errNotContainerizedWorkload = "object is not a containerized workload"
)

// Reconcile event reasons.
const (
	reasonListTraits = "ListedTraits"

	reasonCannotListManualScalerTraits       = "CannotListManualScalerTraits"
	reasonMultipleManualScalerTraits         = "MultipleManualScalerTraits"
	reasonCannotPackageContainerizedWorkload = "CannotPackageContainerizedWorkload"
)

// SetupContainerizedWorkload adds a controller that reconciles ContainerizedWorkloads.
func SetupContainerizedWorkload(mgr ctrl.Manager, l logging.Logger) error {
	name := "oam/" + strings.ToLower(oamv1alpha2.ContainerizedWorkloadGroupKind)

	return ctrl.NewControllerManagedBy(mgr).
		Named(name).
		For(&oamv1alpha2.ContainerizedWorkload{}).
		Complete(workload.NewReconciler(mgr,
			workload.Kind(oamv1alpha2.ContainerizedWorkloadGroupVersionKind),
			workload.WithLogger(l.WithValues("controller", name)),
			workload.WithRecorder(event.NewAPIRecorder(mgr.GetEventRecorderFor(name))),
			workload.WithPackager(workload.PackageFn(containerizedWorkloadPackager))))
}

func containerizedWorkloadPackager(ctx context.Context, app *v1alpha1.KubernetesApplication, obj runtime.Object) error {
	cw, ok := obj.(*oamv1alpha2.ContainerizedWorkload)
	if !ok {
		return errors.New(errNotContainerizedWorkload)
	}

	d := &appsv1.Deployment{}
	if cw.Spec.OperatingSystem != nil {
		d.Spec.Template.Spec.NodeSelector["beta.kubernetes.io/os"] = string(*cw.Spec.OperatingSystem)
	}

	if cw.Spec.CPUArchitecture != nil {
		d.Spec.Template.Spec.NodeSelector["kubernetes.io/arch"] = string(*cw.Spec.CPUArchitecture)
	}

	for _, container := range cw.Spec.Containers {
		if container.ImagePullSecret != nil {
			d.Spec.Template.Spec.ImagePullSecrets = append(d.Spec.Template.Spec.ImagePullSecrets, corev1.LocalObjectReference{
				Name: *container.ImagePullSecret,
			})
		}
		kubernetesContainer := corev1.Container{
			Name:    container.Name,
			Image:   container.Image,
			Command: container.Command,
			Args:    container.Arguments,
			Resources: corev1.ResourceRequirements{
				Requests: corev1.ResourceList{
					corev1.ResourceCPU:    container.Resources.CPU.Required,
					corev1.ResourceMemory: container.Resources.Memory.Required,
				},
			},
		}

		for _, p := range container.Ports {
			kubernetesContainer.Ports = append(kubernetesContainer.Ports, corev1.ContainerPort{
				Name:          p.Name,
				ContainerPort: p.Port,
				Protocol:      corev1.Protocol(*p.Protocol),
			})
		}

		for _, e := range container.Environment {
			kubernetesContainer.Env = append(kubernetesContainer.Env, corev1.EnvVar{
				Name:  e.Name,
				Value: e.Value,
			})
		}

		if container.LivenessProbe != nil {
			kubernetesContainer.LivenessProbe = &corev1.Probe{}
			if container.LivenessProbe.InitialDelaySeconds != nil {
				kubernetesContainer.LivenessProbe.InitialDelaySeconds = *container.LivenessProbe.InitialDelaySeconds
			}
			if container.LivenessProbe.TimeoutSeconds != nil {
				kubernetesContainer.LivenessProbe.TimeoutSeconds = *container.LivenessProbe.TimeoutSeconds
			}
			if container.LivenessProbe.PeriodSeconds != nil {
				kubernetesContainer.LivenessProbe.PeriodSeconds = *container.LivenessProbe.PeriodSeconds
			}
			if container.LivenessProbe.SuccessThreshold != nil {
				kubernetesContainer.LivenessProbe.SuccessThreshold = *container.LivenessProbe.SuccessThreshold
			}
			if container.LivenessProbe.FailureThreshold != nil {
				kubernetesContainer.LivenessProbe.FailureThreshold = *container.LivenessProbe.FailureThreshold
			}

			// NOTE(hasheddan): Kubernetes specifies that only one type of
			// handler should be provided. OAM does not impose that same
			// restriction. We optimistically check all and set whatever is
			// provided.
			if container.LivenessProbe.HTTPGet != nil {
				kubernetesContainer.LivenessProbe.Handler.HTTPGet = &corev1.HTTPGetAction{
					Path: container.LivenessProbe.HTTPGet.Path,
					Port: intstr.IntOrString{IntVal: container.LivenessProbe.HTTPGet.Port},
				}

				for _, h := range container.LivenessProbe.HTTPGet.HTTPHeaders {
					kubernetesContainer.LivenessProbe.Handler.HTTPGet.HTTPHeaders = append(kubernetesContainer.LivenessProbe.Handler.HTTPGet.HTTPHeaders, corev1.HTTPHeader{
						Name:  h.Name,
						Value: h.Value,
					})
				}
			}
			if container.LivenessProbe.Exec != nil {
				kubernetesContainer.LivenessProbe.Exec = &corev1.ExecAction{
					Command: container.LivenessProbe.Exec.Command,
				}
			}
			if container.LivenessProbe.TCPSocket != nil {
				kubernetesContainer.LivenessProbe.TCPSocket = &corev1.TCPSocketAction{
					Port: intstr.IntOrString{IntVal: container.LivenessProbe.TCPSocket.Port},
				}
			}
		}

		if container.ReadinessProbe != nil {
			kubernetesContainer.ReadinessProbe = &corev1.Probe{}
			if container.ReadinessProbe.InitialDelaySeconds != nil {
				kubernetesContainer.ReadinessProbe.InitialDelaySeconds = *container.ReadinessProbe.InitialDelaySeconds
			}
			if container.ReadinessProbe.TimeoutSeconds != nil {
				kubernetesContainer.ReadinessProbe.TimeoutSeconds = *container.ReadinessProbe.TimeoutSeconds
			}
			if container.ReadinessProbe.PeriodSeconds != nil {
				kubernetesContainer.ReadinessProbe.PeriodSeconds = *container.ReadinessProbe.PeriodSeconds
			}
			if container.ReadinessProbe.SuccessThreshold != nil {
				kubernetesContainer.ReadinessProbe.SuccessThreshold = *container.ReadinessProbe.SuccessThreshold
			}
			if container.ReadinessProbe.FailureThreshold != nil {
				kubernetesContainer.ReadinessProbe.FailureThreshold = *container.ReadinessProbe.FailureThreshold
			}

			// NOTE(hasheddan): Kubernetes specifies that only one type of
			// handler should be provided. OAM does not impose that same
			// restriction. We optimistically check all and set whatever is
			// provided.
			if container.ReadinessProbe.HTTPGet != nil {
				kubernetesContainer.ReadinessProbe.Handler.HTTPGet = &corev1.HTTPGetAction{
					Path: container.ReadinessProbe.HTTPGet.Path,
					Port: intstr.IntOrString{IntVal: container.ReadinessProbe.HTTPGet.Port},
				}

				for _, h := range container.ReadinessProbe.HTTPGet.HTTPHeaders {
					kubernetesContainer.ReadinessProbe.Handler.HTTPGet.HTTPHeaders = append(kubernetesContainer.ReadinessProbe.Handler.HTTPGet.HTTPHeaders, corev1.HTTPHeader{
						Name:  h.Name,
						Value: h.Value,
					})
				}
			}
			if container.ReadinessProbe.Exec != nil {
				kubernetesContainer.ReadinessProbe.Exec = &corev1.ExecAction{
					Command: container.ReadinessProbe.Exec.Command,
				}
			}
			if container.ReadinessProbe.TCPSocket != nil {
				kubernetesContainer.ReadinessProbe.TCPSocket = &corev1.TCPSocketAction{
					Port: intstr.IntOrString{IntVal: container.ReadinessProbe.TCPSocket.Port},
				}
			}
		}

		for _, v := range container.Resources.Volumes {
			mount := corev1.VolumeMount{
				Name:      v.Name,
				MountPath: v.MouthPath,
			}
			if v.AccessMode != nil && *v.AccessMode == oamv1alpha2.VolumeAccessModeRO {
				mount.ReadOnly = true
			}
			kubernetesContainer.VolumeMounts = append(kubernetesContainer.VolumeMounts, mount)

		}
		d.Spec.Template.Spec.Containers = append(d.Spec.Template.Spec.Containers, kubernetesContainer)
	}

	return nil
}
