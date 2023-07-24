package scrapeconfigs

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	parcav1alpha1 "github.com/ricoberger/parca-operator/api/v1alpha1"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
	"sigs.k8s.io/yaml"
)

// ParcaConfig holds all the configuration information for Parca.
type ParcaConfig struct {
	ObjectStorage *ParcaObjectStorage          `json:"object_storage,omitempty"`
	ScrapeConfigs []parcav1alpha1.ScrapeConfig `json:"scrape_configs,omitempty"`
}

type ParcaObjectStorage struct {
	Bucket struct {
		Type   string      `json:"type"`
		Config interface{} `json:"config"`
		Prefix string      `json:"prefix" default:""`
	} `json:"bucket,omitempty"`
}

var (
	reconciliationInterval time.Duration
	baseConfig             ParcaConfig
	finalConfig            ParcaConfig
)

// Init is responsible to initialise the scrapeconfigs package. For that it reads all environment variables starting
// with "PARCA_SCRAPECONFIG_" and uses them to configure the package.
func Init() error {
	// Read the "PARCA_SCRAPECONFIG_RECONCILIATION_INTERVAL" environment variable if set. If not set the default value
	// of 5 minutes is used for the reconciliation interval. If the value is seet and can be parsed as a time.Duration
	// the parsed value is used.
	reconciliationInterval = 5 * time.Minute
	if os.Getenv("PARCA_SCRAPECONFIG_RECONCILIATION_INTERVAL") != "" {
		if parsedReconciliationInterval, err := time.ParseDuration(os.Getenv("PARCA_SCRAPECONFIG_RECONCILIATION_INTERVAL")); err == nil {
			reconciliationInterval = parsedReconciliationInterval
		}
	}

	// Read the "PARCA_SCRAPECONFIG_BASE_CONFIG" environment variable. This variable must point to a file which contains
	// the base configuration for Parca. After the file was read we expand all environment variables in the file. Then
	// we unmarshal the YAML content into the "baseConfig" variable.
	baseConfigContent, err := os.ReadFile(os.Getenv("PARCA_SCRAPECONFIG_BASE_CONFIG"))
	if err != nil {
		return err
	}

	baseConfigContent = []byte(expandEnv(string(baseConfigContent)))
	if err := yaml.Unmarshal(baseConfigContent, &baseConfig); err != nil {
		return err
	}

	// Create a new Kubernetes client "c" which is used to interact with the Kubernetes API. This is needed because, we
	// might have an existing final configuration which we need to update.
	restConfig, err := config.GetConfig()
	if err != nil {
		return err
	}

	c, err := client.New(restConfig, client.Options{})
	if err != nil {
		return err
	}

	// Read the existing final configuration from the Kubernetes API. If there is no existing final configuration we use
	// the base configuration as the final configuration and create a new Kubernetes Secret with the content of the base
	// configuration.
	existingConfigSecret := &corev1.Secret{}
	err = c.Get(context.Background(), types.NamespacedName{Name: os.Getenv("PARCA_SCRAPECONFIG_FINAL_CONFIG_NAME"), Namespace: os.Getenv("PARCA_SCRAPECONFIG_FINAL_CONFIG_NAMESPACE")}, existingConfigSecret)
	if err != nil {
		if errors.IsNotFound(err) {
			finalConfig = baseConfig
			err := c.Create(context.Background(), &corev1.Secret{
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

	// If there is an existing final configuration we unmarshal the content of the "parca.yaml" key into the
	// "finalConfig" variable. Then we apply all changes from the base configuration to the final configuration and
	// update the existing Kubernetes Secret with the new content.
	if err := yaml.Unmarshal(existingConfigSecret.Data["parca.yaml"], &finalConfig); err != nil {
		return err
	}

	// TODO: Currently we only replace the object storage configuration, which means that the scrape_configs from the
	// base configuration are only applied the first time. We should find a way to also update the scrape_configs.
	finalConfig.ObjectStorage = baseConfig.ObjectStorage

	finalbaseConfigContent, err := yaml.Marshal(finalConfig)
	if err != nil {
		return err
	}

	err = c.Update(context.Background(), &corev1.Secret{
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

// expandEnv replaces all environment variables in the provided string. The environment variables can be in the form
// `${var}` or `$var`. If the string should contain a `$` it can be escaped via `$$`.
func expandEnv(s string) string {
	os.Setenv("CRANE_DOLLAR", "$")
	return os.ExpandEnv(strings.Replace(s, "$$", "${CRANE_DOLLAR}", -1))
}

// SetScrapeConfig adds or updates a scrape configuration for the provided ParcaScrapeConfig. It returns a list of
// PodIPs which can then be set in the status of the CR.
func SetScrapeConfig(scrapeConfig parcav1alpha1.ParcaScrapeConfig, pods []corev1.Pod) ([]string, error) {
	if scrapeConfig.Spec.ScrapeConfig.JobName == "" {
		scrapeConfig.Spec.ScrapeConfig.JobName = fmt.Sprintf("%s.%s", scrapeConfig.Namespace, scrapeConfig.Name)
	}

	var podIPs []string
	container := getContainerName(scrapeConfig.Spec.Port, pods)

	for _, pod := range pods {
		scrapeConfig.Spec.ScrapeConfig.StaticConfigs = append(scrapeConfig.Spec.ScrapeConfig.StaticConfigs, parcav1alpha1.StaticConfig{
			Targets: []string{fmt.Sprintf("%s:%d", pod.Status.PodIP, scrapeConfig.Spec.Port)},
			Labels: map[string]string{
				"name":      scrapeConfig.Name,
				"namespace": scrapeConfig.Namespace,
				"pod":       pod.Name,
				"container": container,
				"node":      pod.Spec.NodeName,
			},
		})

		podIPs = append(podIPs, fmt.Sprintf("%s:%d", pod.Status.PodIP, scrapeConfig.Spec.Port))
	}

	for i := 0; i < len(finalConfig.ScrapeConfigs); i++ {
		if finalConfig.ScrapeConfigs[i].JobName == scrapeConfig.Spec.ScrapeConfig.JobName {
			finalConfig.ScrapeConfigs[i] = scrapeConfig.Spec.ScrapeConfig
			return podIPs, nil
		}
	}

	finalConfig.ScrapeConfigs = append(finalConfig.ScrapeConfigs, scrapeConfig.Spec.ScrapeConfig)
	return podIPs, nil
}

// getContainerName returns the name of the container for the provided port.
func getContainerName(port int32, pods []corev1.Pod) string {
	for _, pod := range pods {
		for _, container := range pod.Spec.Containers {
			for _, containerPort := range container.Ports {
				if containerPort.ContainerPort == port {
					return container.Name
				}
			}
		}
	}

	return "unknown"
}

// DeleteScrapeConfig deletes the scrape configuration for the provided ParcaScrapeConfig.
func DeleteScrapeConfig(scrapeConfig parcav1alpha1.ParcaScrapeConfig) error {
	if scrapeConfig.Spec.ScrapeConfig.JobName == "" {
		scrapeConfig.Spec.ScrapeConfig.JobName = fmt.Sprintf("%s.%s", scrapeConfig.Namespace, scrapeConfig.Name)
	}

	index := -1
	for i := 0; i < len(finalConfig.ScrapeConfigs); i++ {
		if finalConfig.ScrapeConfigs[i].JobName == scrapeConfig.Spec.ScrapeConfig.JobName {
			index = i
		}
	}

	if index != -1 {
		finalConfig.ScrapeConfigs = append(finalConfig.ScrapeConfigs[:index], finalConfig.ScrapeConfigs[index+1:]...)
		return nil
	}

	return fmt.Errorf("failed to find scrape configuration")
}

// GetConfig returns the current configuration as a YAML byte array.
func GetConfig() ([]byte, error) {
	return yaml.Marshal(finalConfig)
}

// GetReconciliationInterval returns the configured reconciliation interval.
func GetReconciliationInterval() time.Duration {
	return reconciliationInterval
}
