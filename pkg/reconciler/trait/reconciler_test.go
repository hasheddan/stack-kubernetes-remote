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
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/pkg/errors"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"github.com/crossplane/crossplane-runtime/pkg/resource/fake"
	"github.com/crossplane/crossplane-runtime/pkg/test"

	traitfake "github.com/hasheddan/stack-kubernetes-remote/pkg/reconciler/trait/fake"
)

var _ reconcile.Reconciler = &Reconciler{}

func TestReconciler(t *testing.T) {
	type args struct {
		m manager.Manager
		t Kind
		o []ReconcilerOption
	}

	type want struct {
		result reconcile.Result
		err    error
	}

	errBoom := errors.New("boom")

	cases := map[string]struct {
		reason string
		args   args
		want   want
	}{
		"GetTraitError": {
			reason: "Any error (except not found) encountered while getting the resource under reconciliation should be returned.",
			args: args{
				m: &fake.Manager{
					Client: &test.MockClient{MockGet: test.NewMockGetFn(errBoom)},
					Scheme: fake.SchemeWith(&traitfake.Trait{}),
				},
				t: Kind(fake.GVK(&traitfake.Trait{})),
			},
			want: want{err: errors.Wrap(errBoom, errGetTrait)},
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			r := NewReconciler(tc.args.m, tc.args.t, tc.args.o...)
			got, err := r.Reconcile(reconcile.Request{})

			if diff := cmp.Diff(tc.want.err, err, test.EquateErrors()); diff != "" {
				t.Errorf("\nReason: %s\nr.Reconcile(...): -want error, +got error:\n%s", tc.reason, diff)
			}

			if diff := cmp.Diff(tc.want.result, got); diff != "" {
				t.Errorf("\nReason: %s\nr.Reconcile(...): -want, +got:\n%s", tc.reason, diff)
			}
		})
	}
}
