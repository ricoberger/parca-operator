# Parca Operator

The Parca Operator can be used to deploy and operator [Parca](https://www.parca.dev). It can be used to deploy the
[Parca Server](https://www.parca.dev/docs/parca) and [Parca Agent](https://www.parca.dev/docs/parca-agent). Besides that
the Parca Operator can also be used to configure the
[pull-based ingestion](https://www.parca.dev/docs/ingestion#pull-based) via Custom Resource Definitions.

> **Note:** The Parca Operator is work in progress. While I plan to add support for the deployment of the Parca Server
> and Parca Agent, these features are not implemented yet.
>
> This means, that the Parca Operator can currently only be used to configure the pull-based ingestion via the
> ParcaScrapeConfig Custom Resource Definition.

## Installation

The Parca Operator can be installed via Helm:

```sh
helm repo add ricoberger https://ricoberger.github.io/helm-charts
helm repo update

helm upgrade --install parca-operator ricoberger/parca-operator
```

Make sure that you set the following environment variables for the Parca Operator:

- `PARCA_OPERATOR_CONFIG`: The path to the configration file for Parca. This file is used as base for the generated Parca configration and should contain you `object_storage` configuration.
- `PARCA_OPERATOR_CONFIG_NAME`: The name of secret which should be generated. The secret contains a `parca.yaml` key with the generated configuration for Parca.
- `PARCA_OPERATOR_CONFIG_NAMESPACE`: The namespace of secret which should be generated. The secret contains a `parca.yaml` key with the generated configuration for Parca.

## API Reference

### ParcaScrapeConfig

```yaml
apiVersion: parca.ricoberger.de/v1alpha1
kind: ParcaScrapeConfig
metadata:
  name:
  namespace:
spec:
  # selector is the selector for the Pods which should be scraped by Parca.
  selector:
    matchLabels:
    matchExpressions:
  # port is the port of the targets which is used to expose the HTTP endpoints.
  port:
  # scrapeConfig is the scrape configuration as it can be set in the Parca configuration.
  scrapeConfig:
    # job_name is the name of the section in the configurtion. If no job_name is provided, it will be automatically
    # generated based on the name and namespace of the CR: "namespace.name"
    job_name:
    # params is a set of query parameters with which the target is scraped.
    params:
    # scrape_interval defines how frequently to scrape the targets of this scrape config.
    scrape_interval:
    # scrape_timeout defines the timeout for scraping targets of this config.
    scrape_timeout:
    # schema sets the URL scheme with which to fetch metrics from targets.
    scheme:
    # normalized_addresses can be set to true if the addresses returned by the endpoints have already been normalized.
    normalized_addresses:
    # profiling_config defines the profiling config for the targets, see https://www.parca.dev/docs/ingestion#pull-based
    # for more information.
    profiling_config:
    # relabel_configs allows dynamic rewriting of the label set for the targets. See
    # https://prometheus.io/docs/prometheus/latest/configuration/configuration/#relabel_config for more information.
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
  port: 7070
  scrapeConfig:
    scrape_interval: 60s
    scrape_timeout: 45s
```

</details>

## Development

After modifying the `*_types.go` file always run the following command to update the generated code for that resource type:

```sh
make generate
```

The above Makefile target will invoke the [controller-gen](https://sigs.k8s.io/controller-tools) utility to update the
`api/v1alpha1/zz_generated.deepcopy.go` file to ensure our API's Go type definitons implement the `runtime.Object`
interface that all Kind types must implement.

Once the API is defined with spec/status fields and CRD validation markers, the CRD manifests can be generated and
updated with the following command:

```sh
make manifests
```

This Makefile target will invoke controller-gen to generate the CRD manifests at `config/crd/bases/ricoberger.de_vaultsecrets.yaml`.

Deploy the CRD and run the operator locally with the default Kubernetes config file present at `$HOME/.kube/config`:

```sh
export PARCA_OPERATOR_CONFIG=parca.yaml
export PARCA_OPERATOR_CONFIG_NAME=parca-generated
export PARCA_OPERATOR_CONFIG_NAMESPACE=parca

make run
```
