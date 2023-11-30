package spoke

import (
	"context"
	"fmt"
	"time"

	openshiftclientset "github.com/openshift/client-go/config/clientset/versioned"
	openshiftoauthclientset "github.com/openshift/client-go/oauth/clientset/versioned"
	routev1 "github.com/openshift/client-go/route/clientset/versioned"
	"github.com/openshift/library-go/pkg/controller/controllercmd"
	foundationagentapp "github.com/stolostron/multicloud-operators-foundation/cmd/agent/app"
	foundationagentoptions "github.com/stolostron/multicloud-operators-foundation/cmd/agent/app/options"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	clusterclientset "open-cluster-management.io/api/client/cluster/clientset/versioned"
	clusterinformers "open-cluster-management.io/api/client/cluster/informers/externalversions"
	operatorv1 "open-cluster-management.io/api/client/operator/clientset/versioned/typed/operator/v1"
	commonoptions "open-cluster-management.io/ocm/pkg/common/options"
)

func NewKlusterletFeatures(ctx context.Context, scheme *runtime.Scheme, controllerContext *controllercmd.ControllerContext,
	o *commonoptions.AgentOptions) ([]operatorv1.KlusterletFeature, error) {

	// create management kube config
	managementKubeConfig, err := clientcmd.BuildConfigFromFlags("", o.SpokeKubeconfigFile)
	if err != nil {
		return nil, fmt.Errorf("unable to get management cluster kube config %v", err)
	}

	managementClusterKubeClient, err := kubernetes.NewForConfig(managementKubeConfig)
	if err != nil {
		return nil, fmt.Errorf("unable to create management cluster kube client %v", err)
	}

	// load managed client config, the work manager agent may not running in the managed cluster.
	managedClusterConfig := managementKubeConfig

	managedClusterDynamicClient, err := dynamic.NewForConfig(managedClusterConfig)
	if err != nil {
		return nil, fmt.Errorf("unable to create managed cluster dynamic client %v", err)
	}

	managedClusterKubeClient, err := kubernetes.NewForConfig(managedClusterConfig)
	if err != nil {
		return nil, fmt.Errorf("unable to create managed cluster kube client %v", err)
	}
	routeV1Client, err := routev1.NewForConfig(managementKubeConfig)
	if err != nil {
		return nil, fmt.Errorf("new route client config error: %v", err)
	}

	openshiftClient, err := openshiftclientset.NewForConfig(managedClusterConfig)
	if err != nil {
		return nil, fmt.Errorf("unable to create managed cluster openshift config clientset %v", err)
	}

	osOauthClient, err := openshiftoauthclientset.NewForConfig(managedClusterConfig)
	if err != nil {
		return nil, fmt.Errorf("unable to create managed cluster openshift oauth clientset %v", err)
	}

	managedClusterClusterClient, err := clusterclientset.NewForConfig(managedClusterConfig)
	if err != nil {
		return nil, fmt.Errorf("unable to create managed cluster cluster clientset %v", err)
	}

	discoveryClient, err := discovery.NewDiscoveryClientForConfig(managedClusterConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create discovery client %v", err)
	}

	kubeInformerFactory := informers.NewSharedInformerFactory(managedClusterKubeClient, 10*time.Minute)
	clusterInformerFactory := clusterinformers.NewSharedInformerFactory(managedClusterClusterClient, 10*time.Minute)

	foundationOptions := foundationagentoptions.NewAgentOptions()
	//TODO: consolidate args
	foundationOptions.ClusterName = o.SpokeClusterName

	foundation, err := foundationagentapp.NewKlusterletFeature(ctx, scheme, managedClusterConfig, managedClusterDynamicClient, managedClusterKubeClient,
		managementClusterKubeClient, routeV1Client, openshiftClient, osOauthClient, discoveryClient,
		managedClusterClusterClient, kubeInformerFactory, clusterInformerFactory, foundationOptions)
	if err != nil {
		return nil, fmt.Errorf("unable to create klusterlet plugin %v", err)
	}

	// add new plugins here

	plugins := make([]operatorv1.KlusterletFeature, 0)
	plugins = append(plugins, *foundation)

	go kubeInformerFactory.Start(ctx.Done())
	go clusterInformerFactory.Start(ctx.Done())

	return plugins, nil
}
