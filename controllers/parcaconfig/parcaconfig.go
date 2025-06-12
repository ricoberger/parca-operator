// Package parcaconfig manages the configuration of Parca. It works based on a
// user provided base configuration and generates a final config, which can then
// be used by the Parca Server. It dynamically adds, updates and removes scrape
// targets based on the ParcaScrapeConfig CRs.
package parcaconfig

import (
	"context"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"
	"sync"

	parcav1alpha1 "github.com/ricoberger/parca-operator/api/v1alpha1"

	parca "github.com/parca-dev/parca/pkg/config"
	"github.com/prometheus/common/model"
	"github.com/prometheus/prometheus/model/relabel"
	"gopkg.in/yaml.v3"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
)

var (
	kubeClient         client.Client
	finalConfigLock    sync.RWMutex
	finalConfig        CustomParcaConfig
	invalidLabelCharRE = regexp.MustCompile(`[^a-zA-Z0-9_]`)
)

// CustomParcaConfig is the same as "parca.Config" except that we add the
// "KubernetesSDConfigs" field to the "ScrapeConfig" struct. This is needed to
// make it easier to generate the service discovery config for Parca.
type CustomParcaConfig struct {
	ObjectStorage *parca.ObjectStorage       `yaml:"object_storage,omitempty"`
	ScrapeConfigs []*CustomParcaScrapeConfig `yaml:"scrape_configs,omitempty"`
}

type CustomParcaScrapeConfig struct {
	ScrapeConfig        parca.ScrapeConfig              `yaml:",inline"`
	KubernetesSDConfigs []CustomParcaKubernetesSDConfig `yaml:"kubernetes_sd_configs,omitempty"`
}

type CustomParcaKubernetesSDConfig struct {
	Role       string                                  `yaml:"role,omitempty"`
	Namespaces CustomParcaKubernetesSDConfigNamespaces `yaml:"namespaces,omitempty"`
}

type CustomParcaKubernetesSDConfigNamespaces struct {
	Names []string `yaml:"names,omitempty"`
}

// expandEnv replaces all environment variables in the provided string. The
// environment variables can be in the form `${var}` or `$var`. If the string
// should contain a `$` it can be escaped via `$$`.
func expandEnv(s string) string {
	os.Setenv("CRANE_DOLLAR", "$")
	return os.ExpandEnv(strings.ReplaceAll(s, "$$", "${CRANE_DOLLAR}"))
}

func sanitizeLabelName(name string) string {
	return invalidLabelCharRE.ReplaceAllString(name, "_")
}

// Init is responsible to initialise the scrapeconfigs package. For that it
// reads all environment variables starting with "PARCA_SCRAPECONFIG_" and uses
// them to configure the package.
func Init() error {
	finalConfigLock.Lock()
	defer finalConfigLock.Unlock()

	var baseConfig CustomParcaConfig

	// Read the "PARCA_SCRAPECONFIG_BASE_CONFIG" environment variable. This
	// variable must point to a file which contains the base configuration for
	// Parca. After the file was read we expand all environment variables in the
	// file. Then we unmarshal the YAML content into the "baseConfig" variable.
	baseConfigContent, err := os.ReadFile(os.Getenv("PARCA_SCRAPECONFIG_BASE_CONFIG"))
	if err != nil {
		return err
	}

	baseConfigContent = []byte(expandEnv(string(baseConfigContent)))
	if err := yaml.Unmarshal(baseConfigContent, &baseConfig); err != nil {
		return err
	}

	// Create a new Kubernetes client "c" which is used to interact with the
	// Kubernetes API. This is needed because, we might have an existing final
	// configuration which we need to update.
	restConfig, err := config.GetConfig()
	if err != nil {
		return err
	}

	c, err := client.New(restConfig, client.Options{})
	if err != nil {
		return err
	}
	kubeClient = c

	// Read the existing final configuration from the Kubernetes API. If there
	// is no existing final configuration we use the base configuration as the
	// final configuration and create a new Kubernetes Secret with the content
	// of the base configuration.
	existingConfigSecret := &corev1.Secret{}
	err = kubeClient.Get(context.Background(), types.NamespacedName{Name: os.Getenv("PARCA_SCRAPECONFIG_FINAL_CONFIG_NAME"), Namespace: os.Getenv("PARCA_SCRAPECONFIG_FINAL_CONFIG_NAMESPACE")}, existingConfigSecret)
	if err != nil {
		if errors.IsNotFound(err) {
			finalConfig = baseConfig
			err := kubeClient.Create(context.Background(), &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      os.Getenv("PARCA_SCRAPECONFIG_FINAL_CONFIG_NAME"),
					Namespace: os.Getenv("PARCA_SCRAPECONFIG_FINAL_CONFIG_NAMESPACE"),
				},
				Data: map[string][]byte{
					"parca.yaml": baseConfigContent,
				},
			})
			if err != nil {
				return err
			}
			return nil
		}

		return err
	}

	// If there is an existing final configuration we unmarshal the content of
	// the "parca.yaml" key into the "finalConfig" variable. Then we apply all
	// changes from the base configuration to the final configuration and update
	// the existing Kubernetes Secret with the new content.
	if err := yaml.Unmarshal(existingConfigSecret.Data["parca.yaml"], &finalConfig); err != nil {
		return err
	}

	// TODO: Currently we only replace the object storage configuration, which
	// means that the scrape_configs from the base configuration are only
	// applied the first time. We should find a way to also update the
	// scrape_configs.
	finalConfig.ObjectStorage = baseConfig.ObjectStorage

	finalbaseConfigContent, err := yaml.Marshal(finalConfig)
	if err != nil {
		return err
	}

	err = kubeClient.Update(context.Background(), &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      os.Getenv("PARCA_SCRAPECONFIG_FINAL_CONFIG_NAME"),
			Namespace: os.Getenv("PARCA_SCRAPECONFIG_FINAL_CONFIG_NAMESPACE"),
		},
		Data: map[string][]byte{
			"parca.yaml": finalbaseConfigContent,
		},
	})
	if err != nil {
		return err
	}

	return nil
}

// UpdateScrapeConfig adds or updates a scrape configuration for the provided
// ParcaScrapeConfig.
func UpdateScrapeConfig(ctx context.Context, scrapeConfig *parcav1alpha1.ParcaScrapeConfig) error {
	finalConfigLock.Lock()
	defer finalConfigLock.Unlock()

	// Parse the scrape interval and timeout from the ParcaScrapeConfig. Which
	// must be a "model.Duration" type. If the parsing fails we return an error.
	scrapeInterval, err := model.ParseDuration(string(scrapeConfig.Spec.ScrapeConfig.Interval))
	if err != nil {
		return err
	}

	scrapeTimeout, err := model.ParseDuration(string(scrapeConfig.Spec.ScrapeConfig.Timeout))
	if err != nil {
		return err
	}

	// Determine the name of the job as seen by the user. While we are using
	// "namespace/name" as the job name in the scrape config we will replace
	// the "job" label with the value from the CR when provided.
	jobName := scrapeConfig.Spec.ScrapeConfig.Job
	if jobName == "" {
		jobName = fmt.Sprintf("%s/%s", scrapeConfig.Namespace, scrapeConfig.Name)
	}

	// Create the relabel configurations. To use the Kubernetes service
	// discovery we have to add some default relabaling configs. Afterwards we
	// add the relabel configs from the CR.
	relabelConfigs := []*relabel.Config{{
		Action:       relabel.Drop,
		SourceLabels: model.LabelNames{"__meta_kubernetes_pod_phase"},
		Regex:        relabel.MustNewRegexp("(Failed|Succeeded)"),
	}}

	for k, v := range scrapeConfig.Spec.Selector.MatchLabels {
		relabelConfigs = append(relabelConfigs, &relabel.Config{
			Action:       relabel.Keep,
			SourceLabels: model.LabelNames{model.LabelName("__meta_kubernetes_pod_label_" + sanitizeLabelName(k))},
			Regex:        relabel.MustNewRegexp(v),
		})
	}
	for _, exp := range scrapeConfig.Spec.Selector.MatchExpressions {
		switch exp.Operator {
		case metav1.LabelSelectorOpIn:
			relabelConfigs = append(relabelConfigs, &relabel.Config{
				Action:       relabel.Keep,
				SourceLabels: model.LabelNames{model.LabelName("__meta_kubernetes_pod_label_" + sanitizeLabelName(exp.Key))},
				Regex:        relabel.MustNewRegexp(strings.Join(exp.Values, "|")),
			})
		case metav1.LabelSelectorOpNotIn:
			relabelConfigs = append(relabelConfigs, &relabel.Config{
				Action:       relabel.Drop,
				SourceLabels: model.LabelNames{model.LabelName("__meta_kubernetes_pod_label_" + sanitizeLabelName(exp.Key))},
				Regex:        relabel.MustNewRegexp(strings.Join(exp.Values, "|")),
			})
		case metav1.LabelSelectorOpExists:
			relabelConfigs = append(relabelConfigs, &relabel.Config{
				Action:       relabel.Keep,
				SourceLabels: model.LabelNames{model.LabelName("__meta_kubernetes_pod_labelpresent_" + sanitizeLabelName(exp.Key))},
				Regex:        relabel.MustNewRegexp("true"),
			})
		case metav1.LabelSelectorOpDoesNotExist:
			relabelConfigs = append(relabelConfigs, &relabel.Config{
				Action:       relabel.Drop,
				SourceLabels: model.LabelNames{model.LabelName("__meta_kubernetes_pod_labelpresent_" + sanitizeLabelName(exp.Key))},
				Regex:        relabel.MustNewRegexp("true"),
			})
		}
	}

	if scrapeConfig.Spec.ScrapeConfig.Port != "" {
		relabelConfigs = append(relabelConfigs, &relabel.Config{
			Action:       relabel.Keep,
			SourceLabels: model.LabelNames{"__meta_kubernetes_pod_container_port_name"},
			Regex:        relabel.MustNewRegexp(scrapeConfig.Spec.ScrapeConfig.Port),
		})
	} else if scrapeConfig.Spec.ScrapeConfig.PortNumber != 0 {
		relabelConfigs = append(relabelConfigs, &relabel.Config{
			Action:       relabel.Keep,
			SourceLabels: model.LabelNames{"__meta_kubernetes_pod_container_port_number"},
			Regex:        relabel.MustNewRegexp(strconv.FormatInt(scrapeConfig.Spec.ScrapeConfig.PortNumber, 10)),
		})
	}

	relabelConfigs = append(relabelConfigs, &relabel.Config{
		SourceLabels: model.LabelNames{"__meta_kubernetes_namespace"},
		TargetLabel:  "namespace",
	})
	relabelConfigs = append(relabelConfigs, &relabel.Config{
		SourceLabels: model.LabelNames{"__meta_kubernetes_pod_container_name"},
		TargetLabel:  "container",
	})
	relabelConfigs = append(relabelConfigs, &relabel.Config{
		SourceLabels: model.LabelNames{"__meta_kubernetes_pod_name"},
		TargetLabel:  "pod",
	})
	relabelConfigs = append(relabelConfigs, &relabel.Config{
		TargetLabel: "job",
		Replacement: jobName,
	})

	if scrapeConfig.Spec.ScrapeConfig.Port != "" {
		relabelConfigs = append(relabelConfigs, &relabel.Config{
			TargetLabel: "endpoint",
			Replacement: scrapeConfig.Spec.ScrapeConfig.Port,
		})
	} else if scrapeConfig.Spec.ScrapeConfig.PortNumber != 0 {
		relabelConfigs = append(relabelConfigs, &relabel.Config{
			TargetLabel: "endpoint",
			Replacement: strconv.FormatInt(scrapeConfig.Spec.ScrapeConfig.PortNumber, 10),
		})
	}

	for _, relabelConfig := range scrapeConfig.Spec.ScrapeConfig.RelabelConfigs {
		var sourceLabels []model.LabelName
		for _, sourceLabel := range relabelConfig.SourceLabels {
			sourceLabels = append(sourceLabels, model.LabelName(string(sourceLabel)))
		}

		relabelConfigs = append(relabelConfigs, &relabel.Config{
			SourceLabels: sourceLabels,
			Separator:    relabelConfig.Separator,
			TargetLabel:  relabelConfig.TargetLabel,
			Regex:        relabel.MustNewRegexp(relabelConfig.Regex),
			Modulus:      relabelConfig.Modulus,
			Replacement:  relabelConfig.Replacement,
			Action:       relabel.Action(relabelConfig.Action),
		})
	}

	// Create the profiling configuration if it is used in the ParcaScrapeConfig
	// resource. We have to convert our profiling config to the one from Parca,
	// because unfortunately we can not use it directly because it doesn't
	// contain json tags, so that it can not be used in our CRD.
	var profilingConfig *parca.ProfilingConfig

	if scrapeConfig.Spec.ScrapeConfig.ProfilingConfig != nil {
		pprofConfig := make(map[string]*parca.PprofProfilingConfig)

		if scrapeConfig.Spec.ScrapeConfig.ProfilingConfig.PprofConfig != nil {
			for k, v := range scrapeConfig.Spec.ScrapeConfig.ProfilingConfig.PprofConfig {
				var keepSampleType []parca.SampleType
				for _, sampleType := range scrapeConfig.Spec.ScrapeConfig.ProfilingConfig.PprofConfig[k].KeepSampleType {
					keepSampleType = append(keepSampleType, parca.SampleType{
						Type: sampleType.Type,
						Unit: sampleType.Unit,
					})
				}

				pprofConfig[k] = &parca.PprofProfilingConfig{
					Enabled:        v.Enabled,
					Path:           v.Path,
					Delta:          v.Delta,
					KeepSampleType: keepSampleType,
					Seconds:        v.Seconds,
				}
			}
		}

		profilingConfig = &parca.ProfilingConfig{
			PprofConfig: pprofConfig,
			PprofPrefix: scrapeConfig.Spec.ScrapeConfig.ProfilingConfig.PprofPrefix,
		}
	}

	// Create the scrape configuration for the ParcaScrapeConfig as it is
	// required in the Parca configuration file. Afterwards we add or update the
	// corresponding section in the Parca configuration. If we can find an
	// existing scrape configuration (where the job name is "namespace/name") we
	// update this config otherwise a new scrape configuration is added.
	parcaScrapeConfig := CustomParcaScrapeConfig{
		ScrapeConfig: parca.ScrapeConfig{
			JobName:         fmt.Sprintf("%s/%s", scrapeConfig.Namespace, scrapeConfig.Name),
			Params:          scrapeConfig.Spec.ScrapeConfig.Params,
			ScrapeInterval:  scrapeInterval,
			ScrapeTimeout:   scrapeTimeout,
			Scheme:          scrapeConfig.Spec.ScrapeConfig.Scheme,
			ProfilingConfig: profilingConfig,
			RelabelConfigs:  relabelConfigs,
		},
		KubernetesSDConfigs: []CustomParcaKubernetesSDConfig{{
			Role: "pod",
			Namespaces: CustomParcaKubernetesSDConfigNamespaces{
				Names: []string{scrapeConfig.Namespace},
			},
		}},
	}

	finalConfigIndex := -1

	for i := 0; i < len(finalConfig.ScrapeConfigs); i++ {
		if finalConfig.ScrapeConfigs[i].ScrapeConfig.JobName == fmt.Sprintf("%s/%s", scrapeConfig.Namespace, scrapeConfig.Name) {
			finalConfigIndex = i
		}
	}

	if finalConfigIndex != -1 {
		finalConfig.ScrapeConfigs[finalConfigIndex] = &parcaScrapeConfig
	} else {
		finalConfig.ScrapeConfigs = append(finalConfig.ScrapeConfigs, &parcaScrapeConfig)
	}

	// Update the Kubernetes secret with the new value of our configuration.
	// Parca automatically detects the changed configration and will reload it.
	// Afterwards the added / update targets are scraped by Parca.
	finalbaseConfigContent, err := yaml.Marshal(finalConfig)
	if err != nil {
		return err
	}

	err = kubeClient.Update(ctx, &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      os.Getenv("PARCA_SCRAPECONFIG_FINAL_CONFIG_NAME"),
			Namespace: os.Getenv("PARCA_SCRAPECONFIG_FINAL_CONFIG_NAMESPACE"),
		},
		Data: map[string][]byte{
			"parca.yaml": finalbaseConfigContent,
		},
	})
	if err != nil {
		return err
	}

	return nil
}

// DeleteScrapeConfig deletes the scrape configuration for the provided
// ParcaScrapeConfig. The scrape target which should be deleted is determined
// by the job name which must be "namespace/name". If we are not able to find
// such a job we return an error. Otherwise we update the Parca configuration in
// the Kubernetes secret.
func DeleteScrapeConfig(ctx context.Context, scrapeConfig parcav1alpha1.ParcaScrapeConfig) error {
	finalConfigLock.Lock()
	defer finalConfigLock.Unlock()

	finalConfigIndex := -1

	for i := 0; i < len(finalConfig.ScrapeConfigs); i++ {
		if finalConfig.ScrapeConfigs[i].ScrapeConfig.JobName == fmt.Sprintf("%s/%s", scrapeConfig.Namespace, scrapeConfig.Name) {
			finalConfigIndex = i
		}
	}

	if finalConfigIndex != -1 {
		finalConfig.ScrapeConfigs = append(finalConfig.ScrapeConfigs[:finalConfigIndex], finalConfig.ScrapeConfigs[finalConfigIndex+1:]...)

		finalbaseConfigContent, err := yaml.Marshal(finalConfig)
		if err != nil {
			return err
		}

		err = kubeClient.Update(ctx, &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      os.Getenv("PARCA_SCRAPECONFIG_FINAL_CONFIG_NAME"),
				Namespace: os.Getenv("PARCA_SCRAPECONFIG_FINAL_CONFIG_NAMESPACE"),
			},
			Data: map[string][]byte{
				"parca.yaml": finalbaseConfigContent,
			},
		})
		if err != nil {
			return err
		}

		return nil
	}

	return fmt.Errorf("failed to find scrape configuration")
}
