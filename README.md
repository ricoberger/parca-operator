# Parca Operator

The Parca Operator can be used to deploy and operator
[Parca](https://www.parca.dev). It can be used to deploy the
[Parca Server](https://www.parca.dev/docs/parca) and
[Parca Agent](https://www.parca.dev/docs/parca-agent). Besides that the Parca
Operator can also be used to configure the
[pull-based ingestion](https://www.parca.dev/docs/ingestion#pull-based) via
Custom Resource Definitions.

> [!NOTE]
> The Parca Operator is work in progress. While I plan to add support for the
> deployment of the Parca Server and Parca Agent, these features are not
> implemented yet.
>
> This means, that the Parca Operator can currently only be used to configure
> the pull-based ingestion via the ParcaScrapeConfig Custom Resource Definition.

## Installation

The Parca Operator can be installed via Helm:

```sh
helm upgrade --install parca-operator oci://ghcr.io/ricoberger/charts/parca-operator --version <VERSION>
```

Make sure that you set the following environment variables for the Parca
Operator:

- `PARCA_CONFIG_SOURCE`: The path to the configration file for Parca. This file
  is used as source for the generated Parca configration and should contain the
  `object_storage` configuration.
- `PARCA_CONFIG_TARGET_NAME` and `PARCA_CONFIG_TARGET_NAMESPACE`: The name and
  namespace of secret which should be generated. The secret contains a
  `parca.yaml` key with the generated configuration for Parca. The generated
  configuration file contains the content of the source configuration and the
  configuration for all scrape configuration created via a `ParcaScrapeConfig`
  resource.

## API Reference

### ParcaScrapeConfig

```yaml
apiVersion: parca.ricoberger.de/v1alpha1
kind: ParcaScrapeConfig
metadata:
  name:
  namespace:
spec:
  # Selector is the selector for the Pods which should be scraped by Parca.
  selector:
    matchLabels:
    matchExpressions:
  # ScrapeConfig is the scrape configuration as it can be set in the Parca
  # configuration.
  scrapeConfig:
    # Job is the job name of the section in the configurtion. If no job name is
    # provided, it will be automatically generated based on the name and
    # namespace of the CR: "namespace/name"
    job:
    # Port is the name of the port of the Pods which is used to expose the
    # profiling endpoints.
    port:
    # PortNumber is the number of the port which is used to expose the
    # profiling endpoints. This can be used instead of the port field. If the
    # port is not named.
    portNumber:
    # Params is a set of query parameters with which the target is scraped.
    params:
    # Interval defines how frequently to scrape the targets of this scrape
    # config.
    interval:
    # Timeout defines the timeout for scraping targets of this config.
    timeout:
    # Schema sets the URL scheme with which to fetch metrics from targets.
    scheme:
    # ProfilingConfig defines the profiling config for the targets, see
    # https://www.parca.dev/docs/ingestion#pull-based for more information.
    profiling_config:
    # RelabelConfigs allows dynamic rewriting of the label set for the targets,
    # see https://prometheus.io/docs/prometheus/latest/configuration/configuration/#relabel_config
    # for more information.
    relabel_configs:
```

<details>
<summary>Example</summary>

```yaml
apiVersion: parca.ricoberger.de/v1alpha1
kind: ParcaScrapeConfig
metadata:
  name: parca-server
  namespace: parca
spec:
  selector:
    matchLabels:
      app: parca-server
  scrapeConfig:
    port: grpc
    interval: 45s
    timeout: 60s
    profilingConfig:
      pprofConfig:
        fgprof:
          enabled: true
          path: /debug/pprof/fgprof
```

</details>

## Development

After modifying the `*_types.go` file always run the following command to update
the generated code for that resource type:

```sh
make generate
```

The above Makefile target will invoke the
[controller-gen](https://sigs.k8s.io/controller-tools) utility to update the
`api/v1alpha1/zz_generated.deepcopy.go` file to ensure our API's Go type
definitons implement the `runtime.Object` interface that all Kind types must
implement.

Once the API is defined with spec/status fields and CRD validation markers, the
CRD manifests can be generated and updated with the following command:

```sh
make manifests
```

This Makefile target will invoke controller-gen to generate the CRD manifests at
`charts/parca-operator/crds/parca.ricoberger.de_<CRD>.yaml`.

Deploy the CRD and run the operator locally with the default Kubernetes config
file present at `$HOME/.kube/config`:

```sh
export PARCA_CONFIG_SOURCE=parca.yaml
export PARCA_CONFIG_TARGET_NAME=parca-generated
export PARCA_CONFIG_TARGET_NAMESPACE=parca

make run
```
