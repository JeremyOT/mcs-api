# Kubernetes Multi-Cluster Services API

The Multi-Cluster Service API project is being led by [SIG-Multicluster][sig-mc].

This repo hosts the initial implementation according to [KEP-1645][kep] and will
be used for iterative development as we work to meet our Alpha -> Beta
[graduation requirements][grad-reqs].

[sig-mc]: https://github.com/kubernetes/community/tree/master/sig-multicluster
[kep]: https://github.com/kubernetes/enhancements/tree/master/keps/sig-multicluster/1645-multi-cluster-services-api
[grad-reqs]: https://github.com/kubernetes/enhancements/tree/master/keps/sig-multicluster/1645-multi-cluster-services-api#alpha---beta-graduation

## Try it out

_Requires [kind](http://kind.sigs.k8s.io)_

To see the API in action, run `make demo` to build and run a local demo against
a pair of kind clusters. Alternatively, you can take a self guided tour. Use:

- `./demo/up.sh` to create a pair of clusters with mutually connected networks
  and install the `mcs-api-controller`.

  _This will use a pre-existing controller image if available, it's recommended
  to run `make docker-build` first._
- `./demo/demo.sh` to run the same demo as above against your newly created
  clusters (must run `./demo/up.sh` first).
- `./demo/down.sh` to tear down your clusters.

## Community, discussion, contribution, and support

[Our meeting schedule is here][cm].

[cm]: https://github.com/kubernetes/community/tree/master/sig-multicluster#meetings

Our Kubernetes Slack channel is [#sig-multicluster](https://kubernetes.slack.com/messages/sig-multicluster).

## Technical Leads

- @pmorie
- @jeremyot

### Code of conduct

Participation in the Kubernetes community is governed by the [Kubernetes Code of Conduct](code-of-conduct.md).
