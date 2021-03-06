package ksuite

import (
	"context"
	"fmt"
	"github.com/google/uuid"
	"github.com/imdario/mergo"
	"github.com/rancher/k3d/v4/cmd/util"
	k3dcluster "github.com/rancher/k3d/v4/pkg/client"
	"github.com/rancher/k3d/v4/pkg/config"
	"github.com/rancher/k3d/v4/pkg/config/v1alpha2"
	"github.com/rancher/k3d/v4/pkg/runtimes"
	k3d "github.com/rancher/k3d/v4/pkg/types"
	"github.com/rancher/k3d/v4/version"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/suite"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/clientcmd/api"
)

// SuiteStartClusterOpts are start cluster options that are used within the KubeSuite.
var SuiteStartClusterOpts = StartClusterOpts{}

// SuiteStopClusterOpts are stop cluster options that are used within the KubeSuite.
var SuiteStopClusterOpts = StopClusterOpts{}

// KubeSuite a Testify Suite that creates a cluster and contains the config for that cluster. It is up to the user of this library
// to initialize the cluster
//
// For example:
//
type KubeSuite struct {
	suite.Suite

	ClusterConfig *TestClusterConfig
}

func (k *KubeSuite) TearDownSuite() {
	err := StopCluster(k.ClusterConfig, SuiteStopClusterOpts)
	if err != nil {
		k.FailNow(err.Error())
	}
}

func (k *KubeSuite) SetupSuite() {
	var err error
	k.ClusterConfig, err = StartCluster(SuiteStartClusterOpts)
	if err != nil {
		k.FailNow(err.Error())
	}
}

// StartClusterOpts options to start the cluster.
type StartClusterOpts struct {
	// If not set, defaults to the latest K3s version. You can find K3s tags at https://hub.docker.com/r/rancher/k3s/tags.
	K3sImageTag string

	// Creation options that are merged into the default. If not specified, a small K3s server with a single server and no agents is started.
	K3dCreateClusterOpts *v1alpha2.SimpleConfig

	Silent bool
}

type TestClusterConfig struct {
	KubeConfig *api.Config
	RestConfig *rest.Config
	K3dCluster *k3d.Cluster
	Name       string
}

// StartCluster creates a small and ephemeral K3s cluster for testing purposes.
func StartCluster(opts StartClusterOpts) (*TestClusterConfig, error) {
	tag := opts.K3sImageTag
	if tag == "" {
		tag = version.GetK3sVersion(false)
	}
	simpleClusterConfig := v1alpha2.SimpleConfig{
		Name:    uuid.New().String(),
		Image:   fmt.Sprintf("%s:%s", k3d.DefaultK3sImageRepo, tag),
		Servers: 1,
	}

	if opts.Silent {
		logrus.SetLevel(logrus.ErrorLevel)
	}

	if opts.K3dCreateClusterOpts == nil {
		opts.K3dCreateClusterOpts = &v1alpha2.SimpleConfig{}
	}

	_ = mergo.Merge(&simpleClusterConfig, opts.K3dCreateClusterOpts, mergo.WithOverride)

	ctx := context.Background()
	simpleClusterConfig.Options.K3dOptions.Wait = true

	var exposeAPI *k3d.ExposureOpts
	var err error
	if simpleClusterConfig.ExposeAPI.HostPort == "" {
		exposeAPI, err = util.ParsePortExposureSpec("random", k3d.DefaultAPIPort)
		if err != nil {
			return nil, err
		}
	}
	simpleClusterConfig.ExposeAPI = v1alpha2.SimpleExposureOpts{
		Host:     exposeAPI.Host,
		HostIP:   exposeAPI.Binding.HostIP,
		HostPort: exposeAPI.Binding.HostPort,
	}

	clusterConfig, _ := config.TransformSimpleToClusterConfig(ctx, runtimes.SelectedRuntime, simpleClusterConfig)

	clusterConfig, _ = config.ProcessClusterConfig(*clusterConfig)

	if err := k3dcluster.ClusterRun(ctx, runtimes.SelectedRuntime, clusterConfig); err != nil {
		if err := k3dcluster.ClusterDelete(ctx, runtimes.SelectedRuntime, &clusterConfig.Cluster, k3d.ClusterDeleteOpts{SkipRegistryCheck: true}); err != nil {
			return nil, err
		}
		return nil, err
	}

	conf, err := k3dcluster.KubeconfigGet(ctx, runtimes.SelectedRuntime, &clusterConfig.Cluster)
	if err != nil {
		return nil, err
	}

	restConfig, err := clientcmd.NewDefaultClientConfig(*conf, nil).ClientConfig()
	if err != nil {
		return nil, err
	}

	return &TestClusterConfig{
		KubeConfig: conf,
		Name:       clusterConfig.Cluster.Name,
		K3dCluster: &clusterConfig.Cluster,
		RestConfig: restConfig,
	}, nil
}

type StopClusterOpts struct {
}

func StopCluster(cluster *TestClusterConfig, _ StopClusterOpts) error {
	return k3dcluster.ClusterDelete(context.Background(), runtimes.SelectedRuntime, cluster.K3dCluster, k3d.ClusterDeleteOpts{SkipRegistryCheck: true})
}
