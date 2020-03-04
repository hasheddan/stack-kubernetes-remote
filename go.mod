module github.com/hasheddan/stack-kubernetes-remote

go 1.13

replace github.com/crossplane/crossplane => github.com/negz/crossplane v0.1.1-0.20200303070845-e4eccd2f9ad8

require (
	cloud.google.com/go/logging v1.0.0
	github.com/crossplane/crossplane v0.8.0-rc.0.20200303013358-0f1ca6c9e892
	github.com/crossplane/crossplane-runtime v0.5.1-0.20200303062345-185396df417b
	github.com/crossplaneio/crossplane-runtime v0.5.0
	github.com/google/go-cmp v0.3.1
	github.com/pkg/errors v0.8.1
	go.etcd.io/etcd v3.3.18+incompatible
	gopkg.in/alecthomas/kingpin.v2 v2.2.6
	k8s.io/api v0.17.3
	k8s.io/apimachinery v0.17.3
	sigs.k8s.io/controller-runtime v0.4.0
)
