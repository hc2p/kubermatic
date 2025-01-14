/*
Copyright 2023 The Kubermatic Kubernetes Platform contributors.

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

package main

import (
	"context"
	"encoding/base64"
	"flag"
	"fmt"
	"math/rand"
	"net"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackpal/gateway"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v2"

	kubermaticv1 "k8c.io/kubermatic/v2/pkg/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/install/helm"
	"k8c.io/kubermatic/v2/pkg/install/stack"
	kubermaticmaster "k8c.io/kubermatic/v2/pkg/install/stack/kubermatic-master"
	"k8c.io/kubermatic/v2/pkg/log"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	ctrlruntimeconfig "sigs.k8s.io/controller-runtime/pkg/client/config"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

const (
	nip   = "nip.io"
	sslip = "sslip.io"
)

type LocalOptions struct {
	Options

	HelmBinary  string
	HelmTimeout time.Duration
	Endpoint    string
}

func LocalCommand(logger *logrus.Logger) *cobra.Command {
	opt := LocalOptions{
		HelmTimeout: 5 * time.Minute,
		HelmBinary:  "helm",
		Options: Options{
			ChartsDirectory: "./charts",
		},
	}
	cmd := &cobra.Command{
		Use:   "local [environment]",
		Short: "Initialize environment for simplified local KKP installation",
		Long:  "Prepares minimal Kubernetes environment (e.g. kind) and auto-configures a non-production KKP installation for evaluation and development purpose.",
		PreRun: func(cmd *cobra.Command, args []string) {
			options.CopyInto(&opt.Options)
			if opt.HelmBinary == "" {
				opt.HelmBinary = os.Getenv("HELM_BINARY")
			}
		},
	}
	cmd.PersistentFlags().DurationVar(&opt.HelmTimeout, "helm-timeout", opt.HelmTimeout, "time to wait for Helm operations to finish")
	cmd.PersistentFlags().StringVar(&opt.HelmBinary, "helm-binary", opt.HelmBinary, "full path to the Helm 3 binary to use")
	cmd.PersistentFlags().StringVar(&opt.Endpoint, "endpoint", "", "endpoint address for KKP installation (e.g. 10.0.0.5.nip.io), if this flag is left empty, the installer does best effort in auto-configuring from available network interfaces")

	cmd.AddCommand(localKindCommand(logger, opt))
	// TODO: expose when ready
	cmd.Hidden = true
	return cmd
}

func localKindCommand(logger *logrus.Logger, opt LocalOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "kind",
		Short: "Initialize kind environment for simplified local KKP installation",
		Long:  "Prepares minimal kind environment and auto-configures a non-production KKP installation for evaluation and development purpose.",
		PreRun: func(cmd *cobra.Command, args []string) {
			_, err := exec.LookPath("kind")
			if err != nil {
				logger.Fatalf("failed to find 'kind' binary: %v", err)
			}
			_, err = exec.LookPath("helm")
			if err != nil {
				logger.Fatalf("failed to find 'helm' binary: %v", err)
			}
		},
		RunE: localKindFunc(logger, opt),
	}
	return cmd
}

func localKind(logger *logrus.Logger, dir string) (ctrlruntimeclient.Client, context.CancelFunc) {
	kindConfig := filepath.Join(dir, "kind-config.yaml")
	if err := os.WriteFile(kindConfig, []byte(kindConfigContent), 0600); err != nil {
		logger.Fatalf("failed to create 'kind' config: %v", err)
	}
	logger.Info("Creating kind cluster ...")
	// TODO: make this idempotent
	out, err := exec.Command("kind", "create", "cluster", "-n", "kkp-cluster", "--config", kindConfig).CombinedOutput()
	if err != nil {
		logger.Fatalf("failed to create 'kind' cluster: %v\n%v", err, string(out))
	}

	logger.Info("Kind cluster ready, continuing configuration ...")
	kubeconfigCmd := exec.Command("kubectl", "config", "view", "--minify", "--flatten")
	kindKubeConfigPath := filepath.Join(dir, "kube-config.yaml")
	kindKubeConfig, err := os.Create(kindKubeConfigPath)
	if err != nil {
		logger.Fatalf("failed to create 'kind' cluster kubeconfig: %v", err)
	}
	kubeconfigCmd.Stdout = kindKubeConfig
	if err = kubeconfigCmd.Run(); err != nil {
		logger.Fatalf("failed to write 'kind' cluster kubeconfig: %v", err)
	}
	if err := kindKubeConfig.Close(); err != nil {
		logger.Fatalf("failed to close 'kind' cluster kubeconfig: %v", err)
	}

	if err := flag.Set("kubeconfig", kindKubeConfigPath); err != nil {
		logger.Fatalf("failed to close set kubeconfig path: %v", err)
	}
	ctrlConfig, err := ctrlruntimeconfig.GetConfig()
	if err != nil {
		logger.Fatalf("failed to initialize runtime config: %v", err)
	}

	mgr, err := manager.New(ctrlConfig, manager.Options{
		MetricsBindAddress:     "0",
		HealthProbeBindAddress: "0",
	})
	if err != nil {
		logger.Fatalf("failed to construct mgr: %v", err)
	}

	// start the manager in its own goroutine
	appContext := context.Background()

	go func() {
		if err := mgr.Start(appContext); err != nil {
			logger.Fatalf("Failed to start Kubernetes client manager: %v", err)
		}
	}()

	// wait for caches to be synced
	mgrSyncCtx, cancel := context.WithTimeout(appContext, 30*time.Second)
	defer cancel()
	if synced := mgr.GetCache().WaitForCacheSync(mgrSyncCtx); !synced {
		logger.Fatal("Timed out while waiting for Kubernetes client caches to synchronize.")
	}
	return mgr.GetClient(), cancel
}

func ensureResource(kubeClient ctrlruntimeclient.Client, logger *logrus.Logger, o ctrlruntimeclient.Object) {
	if err := kubeClient.Create(context.Background(), o); err != nil && !apierrors.IsAlreadyExists(err) {
		logger.Fatal(err)
	}
}

func installKubevirt(logger *logrus.Logger, dir string, helmClient helm.Client, opt LocalOptions) {
	logger.Info("Installing KubeVirt ...")
	err := helmClient.InstallChart("", "kubevirt", filepath.Join(opt.ChartsDirectory, "local-kubevirt"), "", nil, nil)
	if err != nil {
		logger.Fatalf("Failed to install KubeVirt Helm client: %v", err)
	}
}

func installKubermatic(logger *logrus.Logger, dir string, kubeClient ctrlruntimeclient.Client, helmClient helm.Client, opts LocalOptions) string {
	kkpEndpoint := getLocalEndpoint(logger, opts)
	logger.Infof("Installing KKP at %v ...", kkpEndpoint) // TODO: prettify
	valuesExamplePath := filepath.Join(dir, "values.example.yaml")
	kubermaticExamplePath := filepath.Join(dir, "kubermatic.example.yaml")
	valuesPath := filepath.Join(dir, "values.yaml")
	kubermaticPath := filepath.Join(dir, "kubermatic.yaml")

	_, uk, err := loadKubermaticConfiguration(kubermaticExamplePath)
	if err != nil {
		logger.Fatalf("failed to unmarshal example kubermatic.yaml: %v", err)
	}
	// TODO: make this idempotent (e.g. if generated yamls already exist, don't touch them)
	uSetNestedField(logger, uk.Object, kkpEndpoint, "spec", "ingress", "domain")
	uSetNestedField(logger, uk.Object, nil, "spec", "ingress", "certificateIssuer")
	uSetNestedField(logger, uk.Object, fmt.Sprintf("http://%v/dex", kkpEndpoint), "spec", "auth", "tokenIssuer")
	uSetNestedField(logger, uk.Object, randomString(32), "spec", "auth", "issuerClientSecret")
	uSetNestedField(logger, uk.Object, randomString(32), "spec", "auth", "issuerCookieKey")
	uSetNestedField(logger, uk.Object, randomString(32), "spec", "auth", "serviceAccountKey")
	kout, err := yaml.Marshal(uk.Object)
	if err != nil {
		logger.Fatalf("failed to marshal kubermatic.yaml: %v", err)
	}
	if err := os.WriteFile(kubermaticPath, kout, 0600); err != nil {
		logger.Fatalf("failed to create kubermatic.yaml: %v", err)
	}

	vc, err := os.ReadFile(valuesExamplePath)
	if err != nil {
		logger.Fatalf("failed to read values.yaml example: %v", err)
	}
	uv := make(map[any]any)
	if err := yaml.Unmarshal(vc, uv); err != nil {
		logger.Fatalf("failed to decode example values.yaml: %v", err)
	}

	setNestedField(logger, uv, "http", "dex", "ingress", "scheme")
	setNestedField(logger, uv, kkpEndpoint, "dex", "ingress", "host")
	clients := uv["dex"].(map[any]any)["clients"].([]any)
	for i := range clients {
		clientsMap := clients[i].(map[any]any)
		setNestedField(logger, clientsMap, randomString(32), "secret")
		uris := clientsMap["RedirectURIs"].([]any)
		for uri := range uris {
			u, err := url.Parse(uris[uri].(string))
			if err != nil {
				logger.Fatalf("failed to modify values.yaml: %v", err)
			}
			u.Scheme = "http"
			u.Host = kkpEndpoint
			uris[uri] = u.String()
		}
	}

	setNestedField(logger, uv, uuid.NewString(), "telemetry", "uuid")
	setNestedField(logger, uv, nil, "minio")
	vout, err := yaml.Marshal(uv)
	if err != nil {
		logger.Fatalf("failed to marshal values.yaml: %v", err)
	}
	if err := os.WriteFile(valuesPath, vout, 0600); err != nil {
		logger.Fatalf("failed to create values.yaml: %v", err)
	}

	ensureResource(kubeClient, logger, &kindIngressControllerNamespace)
	ensureResource(kubeClient, logger, &kindKubermaticNamespace)
	ensureResource(kubeClient, logger, &kindStorageClass)
	ensureResource(kubeClient, logger, &kindIngressControllerService)
	ensureResource(kubeClient, logger, &kindNodeportProxyService)

	ms := kubermaticmaster.MasterStack{}
	k, uk, err := loadKubermaticConfiguration(kubermaticPath)
	if err != nil {
		logger.Panicf("Failed to load %v after autoconfiguration: %v", kubermaticPath, err)
	}
	v, err := loadHelmValues(valuesPath)
	if err != nil {
		logger.Panicf("Failed to load %v after autoconfiguration: %v", valuesPath, err)
	}
	msOpts := stack.DeployOptions{
		ChartsDirectory:            opts.ChartsDirectory,
		KubeClient:                 kubeClient,
		HelmClient:                 helmClient,
		Logger:                     log.Prefix(logrus.NewEntry(logger), "   "),
		MLASkipMinio:               true,
		MLASkipMinioLifecycleMgr:   true,
		KubermaticConfiguration:    k,
		RawKubermaticConfiguration: uk,
		HelmValues:                 v,
	}

	if err := ms.Deploy(context.Background(), msOpts); err != nil {
		logger.Fatalf("Failed to deploy kubermatic stack: %v", err)
	}

	kubeconfig := filepath.Join(dir, "kube-config.yaml")
	internalKubeconfig := filepath.Join(dir, "kube-config-internal.yaml")
	kindSeedSecret := initKindSeedSecret(kubeClient, logger, kubeconfig, internalKubeconfig)
	ensureResource(kubeClient, logger, &kindSeedSecret)
	kindPreset := initKindPreset(logger, internalKubeconfig)
	ensureResource(kubeClient, logger, &kindPreset)
	ensureResource(kubeClient, logger, &kindLocalSeed)
	return kkpEndpoint
}

func localKindFunc(logger *logrus.Logger, opt LocalOptions) cobraFuncE {
	return handleErrors(logger, func(cmd *cobra.Command, args []string) error {
		dir := "./examples"
		path, err := os.Executable()
		if err == nil {
			dir = filepath.Join(filepath.Dir(path), "examples")
		}
		kubeClient, cancel := localKind(logger, dir)
		defer cancel()
		kubeconfig := filepath.Join(dir, "kube-config.yaml")
		helmClient, err := helm.NewCLI(opt.HelmBinary, kubeconfig, "", opt.HelmTimeout, logger)
		if err != nil {
			logger.Fatalf("Failed to create helm client: %v", err)
		}
		helmVersion, err := helmClient.Version()
		if err != nil {
			logger.Fatalf("Failed to check Helm version: %v", err)
		}
		if helmVersion.LessThan(MinHelmVersion) {
			logger.Fatalf(
				"the installer requires Helm >= %s, but detected %q as %s (use --helm-binary or $HELM_BINARY to override)",
				MinHelmVersion,
				opt.HelmBinary,
				helmVersion,
			)
		}
		installKubevirt(logger, dir, helmClient, opt)
		endpoint := installKubermatic(logger, dir, kubeClient, helmClient, opt)
		logger.Infof("KKP installed successfully, login at http://%v", endpoint)
		return nil
	})
}

func getLocalEndpoint(logger *logrus.Logger, opts LocalOptions) string {
	if opts.Endpoint != "" {
		if ip := net.ParseIP(opts.Endpoint); ip != nil {
			return ipToNip(ip)
		}
		return opts.Endpoint
	}
	if gwip, err := gateway.DiscoverGateway(); err != nil {
		logger.Fatalf("Failed to determine default gateway IP, please use --endpoint flag: %v", err)
	} else {
		addrs, err := net.InterfaceAddrs()
		if err != nil {
			logger.Fatalf("Failed to list interface addresses, please use --endpoint flag: %v", err)
		}
		for _, a := range addrs {
			if ipnet, ok := a.(*net.IPNet); ok && !ipnet.IP.IsLoopback() && ipnet.Contains(gwip) {
				return ipToNip(ipnet.IP)
			}
		}
	}
	logger.Fatalf("Failed to determine local IP, please use --endpoint flag")
	return ""
}

func ipToNip(ip net.IP) string {
	if ip.To4() != nil {
		return fmt.Sprintf("%v.%v", ip.String(), nip)
	}
	// nip.io is much faster and more reliable but doesn't support IPv6
	processedIP := strings.ReplaceAll(ip.String(), ":", "-")
	return fmt.Sprintf("%v.%v", processedIP, sslip)
}

func uSetNestedField(logger *logrus.Logger, obj map[string]any, value any, fields ...string) {
	if err := unstructured.SetNestedField(obj, value, fields...); err != nil {
		logger.Fatalf("failed to set path %v: %v", fields, err)
	}
}

func setNestedField(logger *logrus.Logger, obj map[any]any, value any, fields ...string) {
	m := obj
	for _, field := range fields[:len(fields)-1] {
		if val, ok := m[field]; ok {
			if valMap, ok := val.(map[any]any); ok {
				m = valMap
			} else {
				logger.Fatalf("value cannot be set because %v is not a map[any]any: %T %v, path %v", field, val, val, fields)
			}
		} else {
			newVal := make(map[any]any)
			m[field] = newVal
			m = newVal
		}
	}
	m[fields[len(fields)-1]] = value
}

func initKindSeedSecret(kubeClient ctrlruntimeclient.Client, logger *logrus.Logger, kubeconfigPath, internalKubeconfigPath string) corev1.Secret {
	cpPod := corev1.Pod{}
	key := ctrlruntimeclient.ObjectKey{Namespace: "kube-system", Name: "kube-apiserver-kkp-cluster-control-plane"}
	if err := kubeClient.Get(context.Background(), key, &cpPod); err != nil {
		logger.Fatalf("Failed to get IP for kind control-plane pod: %v", err)
	}
	ip := cpPod.Status.PodIP
	k, err := os.ReadFile(kubeconfigPath)
	if err != nil {
		logger.Fatalf("Failed to initialize seed secret: %v", err)
	}
	addrRe := regexp.MustCompile(`([ ]*server:) https://127.0.0.1:[0-9]*`)
	internalKubeconfig := addrRe.ReplaceAllString(string(k), fmt.Sprintf(`$1 https://%v:6443`, ip))

	if err := os.WriteFile(internalKubeconfigPath, []byte(internalKubeconfig), 0600); err != nil {
		logger.Fatalf("failed to write internal kubeconfig: %v", err)
	}
	kindKubeconfigSeedSecret.StringData["kubeconfig"] = internalKubeconfig
	return kindKubeconfigSeedSecret
}

func initKindPreset(logger *logrus.Logger, internalKubeconfigPath string) kubermaticv1.Preset {
	k, err := os.ReadFile(internalKubeconfigPath)
	if err != nil {
		logger.Fatalf("Failed to initialize preset: %v", err)
	}
	kindLocalPreset.Spec.Kubevirt.Kubeconfig = base64.StdEncoding.EncodeToString(k)
	return kindLocalPreset
}

func randomString(n int) string {
	var chars = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0987654321")
	str := make([]rune, n)
	for i := range str {
		str[i] = chars[rand.Intn(len(chars))]
	}
	return string(str)
}
