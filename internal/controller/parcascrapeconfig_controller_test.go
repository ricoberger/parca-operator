package controller

import (
	"context"
	"os"
	"path/filepath"

	parcav1alpha1 "github.com/ricoberger/parca-operator/api/v1alpha1"
	"github.com/ricoberger/parca-operator/internal/parcaconfig"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"gopkg.in/yaml.v3"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

var _ = Describe("ParcaScrapeConfig Controller", func() {
	Context("When reconciling a resource", func() {
		ctx := context.Background()

		typeNamespacedName := types.NamespacedName{
			Name:      "test",
			Namespace: "default",
		}
		parcascrapeconfig := &parcav1alpha1.ParcaScrapeConfig{}

		BeforeEach(func() {
			By("Set environment variables")
			os.Setenv("PARCA_CONFIG_SOURCE", filepath.Join("tests", "source-config.yaml"))
			os.Setenv("PARCA_CONFIG_TARGET_NAME", "parca")
			os.Setenv("PARCA_CONFIG_TARGET_NAMESPACE", "default")

			By("Init Parca configuration")
			err := parcaconfig.Init(k8sClient)
			Expect(err).NotTo(HaveOccurred())

			By("Creating the custom resource for the Kind ParcaScrapeConfig")
			err = k8sClient.Get(ctx, typeNamespacedName, parcascrapeconfig)
			if err != nil && errors.IsNotFound(err) {
				pprofConfigEnabled := true

				resource := &parcav1alpha1.ParcaScrapeConfig{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test",
						Namespace: "default",
					},
					Spec: parcav1alpha1.ParcaScrapeConfigSpec{
						Selector: metav1.LabelSelector{
							MatchLabels: map[string]string{"app": "test"},
						},
						ScrapeConfig: parcav1alpha1.ScrapeConfig{
							Port:     "http",
							Interval: "45s",
							Timeout:  "60s",
							ProfilingConfig: &parcav1alpha1.ProfilingConfig{
								PprofConfig: parcav1alpha1.PprofConfig{
									"fgprof": &parcav1alpha1.PprofProfilingConfig{
										Enabled: &pprofConfigEnabled,
										Path:    "/debug/pprof/fgprof",
									},
								},
							},
						},
					},
				}
				Expect(k8sClient.Create(ctx, resource)).To(Succeed())
			}
		})

		AfterEach(func() {
			resource := &parcav1alpha1.ParcaScrapeConfig{}
			err := k8sClient.Get(ctx, typeNamespacedName, resource)
			Expect(err).NotTo(HaveOccurred())

			By("Cleanup the specific resource instance ParcaScrapeConfig")
			Expect(k8sClient.Delete(ctx, resource)).To(Succeed())
		})

		It("Should successfully reconcile the resource", func() {
			By("Reconciling the created resource")
			controllerReconciler := &ParcaScrapeConfigReconciler{
				Client: k8sClient,
				Scheme: k8sClient.Scheme(),
			}

			_, err := controllerReconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: typeNamespacedName,
			})
			Expect(err).NotTo(HaveOccurred())

			By("Get actual target configuration")
			configSecret := &corev1.Secret{}
			err = k8sClient.Get(ctx, types.NamespacedName{Name: "parca", Namespace: "default"}, configSecret)
			Expect(err).NotTo(HaveOccurred())

			var actualConfig parcaconfig.CustomParcaConfig
			err = yaml.Unmarshal(configSecret.Data["parca.yaml"], &actualConfig)
			Expect(err).NotTo(HaveOccurred())

			By("Load expected target configuration")
			expectedConfigContent, err := os.ReadFile(filepath.Join("tests", "target-config.yaml"))
			Expect(err).NotTo(HaveOccurred())

			var expectedConfig parcaconfig.CustomParcaConfig
			err = yaml.Unmarshal(expectedConfigContent, &expectedConfig)
			Expect(err).NotTo(HaveOccurred())

			By("Compare actual and expected configuration")
			Expect(actualConfig).To(Equal(expectedConfig))
		})
	})
})
