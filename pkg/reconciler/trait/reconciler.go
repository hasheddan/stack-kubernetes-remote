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

package trait

import (
	"context"
	"time"

	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"github.com/crossplane/crossplane-runtime/apis/core/v1alpha1"
	"github.com/crossplane/crossplane-runtime/pkg/event"
	"github.com/crossplane/crossplane-runtime/pkg/logging"
	"github.com/crossplane/crossplane-runtime/pkg/resource"
	workloadv1alpha1 "github.com/crossplane/crossplane/apis/workload/v1alpha1"
)

const (
	reconcileTimeout = 1 * time.Minute
	shortWait        = 30 * time.Second
	longWait         = 1 * time.Minute
)

// Reconcile error strings.
const (
	errGetTrait               = "cannot get trait"
	errUpdateTraitStatus      = "cannot update trait status"
	errTraitModify            = "cannot apply trait modification"
	errGetPackage             = "cannot get KubernetesApplication for workload reference in trait"
	errApplyTraitModification = "cannot apply trait modification to KubernetesApplication"
)

// Reconcile event reasons.
const (
	reasonTraitModify = "PackageModified"

	reasonCannotGetPackage        = "CannotGetReferencedWorkloadPackage"
	reasonCannotModifyPackage     = "CannotModifyPackage"
	reasonCannotApplyModification = "CannotApplyModification"
)

// A ReconcilerOption configures a Reconciler.
type ReconcilerOption func(*Reconciler)

// WithLogger specifies how the Reconciler should log messages.
func WithLogger(l logging.Logger) ReconcilerOption {
	return func(r *Reconciler) {
		r.log = l
	}
}

// WithRecorder specifies how the Reconciler should record events.
func WithRecorder(er event.Recorder) ReconcilerOption {
	return func(r *Reconciler) {
		r.record = er
	}
}

// WithModifier specifies how the Reconciler should modify the workload package.
func WithModifier(m Modifier) ReconcilerOption {
	return func(r *Reconciler) {
		r.modifier = m
	}
}

// A Reconciler reconciles OAM traits by modifying the the KubernetesApplication
// that packages their referenced workload.
type Reconciler struct {
	client   client.Client
	modifier Modifier
	newTrait func() Trait

	log    logging.Logger
	record event.Recorder
}

// Kind is an OAM trait kind.
type Kind schema.GroupVersionKind

// A Trait is a type of OAM trait.
type Trait interface {
	resource.Conditioned
	metav1.Object
	runtime.Object
	GetWorkloadRef() corev1.ObjectReference
}

// NewReconciler returns a Reconciler that reconciles OAM traits by fetching
// their referenced workload's KubernetesApplication and applying modifications.
func NewReconciler(m ctrl.Manager, trait Kind, o ...ReconcilerOption) *Reconciler {
	nt := func() Trait {
		return resource.MustCreateObject(schema.GroupVersionKind(trait), m.GetScheme()).(Trait)
	}

	r := &Reconciler{
		client:   m.GetClient(),
		modifier: ModifyFn(noopModifier),
		newTrait: nt,

		log:    logging.NewNopLogger(),
		record: event.NewNopRecorder(),
	}

	for _, ro := range o {
		ro(r)
	}

	return r
}

// Reconcile an OAM trait type by modifying its referenced workload's
// KubernetesApplication.
func (r *Reconciler) Reconcile(req reconcile.Request) (reconcile.Result, error) {
	log := r.log.WithValues("request", req)
	log.Debug("Reconciling")

	ctx, cancel := context.WithTimeout(context.Background(), reconcileTimeout)
	defer cancel()

	trait := r.newTrait()
	if err := r.client.Get(ctx, req.NamespacedName, trait); err != nil {
		return reconcile.Result{}, errors.Wrap(resource.IgnoreNotFound(err), errGetTrait)
	}

	log = log.WithValues("uid", trait.GetUID(), "version", trait.GetResourceVersion())

	app := &workloadv1alpha1.KubernetesApplication{}
	if err := r.client.Get(ctx, types.NamespacedName{Name: trait.GetWorkloadRef().Name, Namespace: trait.GetNamespace()}, app); resource.IgnoreNotFound(err) != nil {
		log.Debug("Cannot get workload package", "error", err, "requeue-after", time.Now().Add(shortWait))
		r.record.Event(trait, event.Warning(reasonCannotGetPackage, err))
		trait.SetConditions(v1alpha1.ReconcileError(errors.Wrap(err, errGetPackage)))
		return reconcile.Result{RequeueAfter: shortWait}, errors.Wrap(r.client.Status().Update(ctx, trait), errUpdateTraitStatus)
	}

	if err := r.modifier.Modify(ctx, app, trait); err != nil {
		log.Debug("Cannot package workload", "error", err, "requeue-after", time.Now().Add(shortWait))
		r.record.Event(trait, event.Warning(reasonCannotModifyPackage, err))
		trait.SetConditions(v1alpha1.ReconcileError(errors.Wrap(err, errTraitModify)))
		return reconcile.Result{RequeueAfter: shortWait}, errors.Wrap(r.client.Status().Update(ctx, trait), errUpdateTraitStatus)
	}

	app.SetOwnerReferences(trait.GetOwnerReferences())
	app.SetName(trait.GetWorkloadRef().Name)
	app.SetNamespace(trait.GetNamespace())
	if err := resource.Apply(ctx, r.client, app, resource.ControllersMustMatch()); err != nil {
		log.Debug("Cannot apply workload package", "error", err, "requeue-after", time.Now().Add(shortWait))
		r.record.Event(trait, event.Warning(reasonCannotApplyModification, err))
		trait.SetConditions(v1alpha1.ReconcileError(errors.Wrap(err, errApplyTraitModification)))
		return reconcile.Result{RequeueAfter: shortWait}, errors.Wrap(r.client.Status().Update(ctx, trait), errUpdateTraitStatus)
	}

	r.record.Event(trait, event.Normal(reasonTraitModify, "Successfully modifed workload package"))
	log.Debug("Successfully modified referenced workload's KubernetesApplication", "kind", trait.GetObjectKind().GroupVersionKind().String())

	trait.SetConditions(v1alpha1.ReconcileSuccess())
	return reconcile.Result{RequeueAfter: longWait}, errors.Wrap(r.client.Status().Update(ctx, trait), errUpdateTraitStatus)
}
