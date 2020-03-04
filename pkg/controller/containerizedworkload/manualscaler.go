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

	"github.com/pkg/errors"
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"

	"github.com/crossplane/crossplane-runtime/pkg/event"
	"github.com/crossplane/crossplane-runtime/pkg/logging"
	oamv1alpha2 "github.com/crossplane/crossplane/apis/oam/v1alpha2"
	"github.com/crossplane/crossplane/apis/workload/v1alpha1"

	"github.com/hasheddan/stack-kubernetes-remote/pkg/reconciler/trait"
)

const (
	errNotManualScalerTrait = "trait is not a manual scaler"
)

// SetupManualScalerTrait adds a controller that reconciles ManualScalers that
// reference a ContainerizedWorkload.
func SetupManualScalerTrait(mgr ctrl.Manager, l logging.Logger) error {
	name := "oam/" + strings.ToLower(oamv1alpha2.ManualScalerTraitGroupKind)

	return ctrl.NewControllerManagedBy(mgr).
		Named(name).
		For(&oamv1alpha2.ManualScalerTrait{}).
		Complete(trait.NewReconciler(mgr,
			trait.Kind(oamv1alpha2.ManualScalerTraitGroupVersionKind),
			trait.WithLogger(l.WithValues("controller", name)),
			trait.WithRecorder(event.NewAPIRecorder(mgr.GetEventRecorderFor(name))),
			trait.WithModifier(trait.ModifyFn(manualScalerModifier))))
}

func manualScalerModifier(ctx context.Context, app *v1alpha1.KubernetesApplication, obj runtime.Object) error {
	ms, ok := obj.(*oamv1alpha2.ManualScalerTrait)
	if !ok {
		return errors.New(errNotManualScalerTrait)
	}

	for i, t := range app.Spec.ResourceTemplates {
		if t.Spec.Template.GetKind() == "Deployment" {
			d := &appsv1.Deployment{}
			if err := runtime.DefaultUnstructuredConverter.FromUnstructured(t.Spec.Template.UnstructuredContent(), d); err != nil {
				return err
			}
			d.Spec.Replicas = &ms.Spec.ReplicaCount
			u, err := runtime.DefaultUnstructuredConverter.ToUnstructured(d)
			if err != nil {
				return err
			}
			app.Spec.ResourceTemplates[i].Spec.Template = &unstructured.Unstructured{Object: u}
		}
	}
	return nil
}
