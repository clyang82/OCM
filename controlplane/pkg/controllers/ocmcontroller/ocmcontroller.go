package ocmcontroller

import (
	"context"
	"net/http"
	_ "net/http/pprof"
	"time"

	"github.com/openshift/library-go/pkg/controller/controllercmd"
	"github.com/openshift/library-go/pkg/controller/factory"
	"github.com/openshift/library-go/pkg/operator/events"
	"github.com/pkg/errors"
	kubeinformers "k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	kubeevents "k8s.io/client-go/tools/events"
	"k8s.io/klog/v2"

	addonclient "open-cluster-management.io/api/client/addon/clientset/versioned"
	addoninformers "open-cluster-management.io/api/client/addon/informers/externalversions"
	clusterv1client "open-cluster-management.io/api/client/cluster/clientset/versioned"
	clusterscheme "open-cluster-management.io/api/client/cluster/clientset/versioned/scheme"
	clusterv1informers "open-cluster-management.io/api/client/cluster/informers/externalversions"
	workv1client "open-cluster-management.io/api/client/work/clientset/versioned"
	workv1informers "open-cluster-management.io/api/client/work/informers/externalversions"
	ocmfeature "open-cluster-management.io/api/feature"
	confighub "open-cluster-management.io/ocm-controlplane/config/hub"
	scheduling "open-cluster-management.io/placement/pkg/controllers/scheduling"
	"open-cluster-management.io/placement/pkg/debugger"
	"open-cluster-management.io/registration/pkg/features"
	"open-cluster-management.io/registration/pkg/helpers"
	"open-cluster-management.io/registration/pkg/hub/addon"
	"open-cluster-management.io/registration/pkg/hub/clusterrole"
	"open-cluster-management.io/registration/pkg/hub/csr"
	"open-cluster-management.io/registration/pkg/hub/lease"
	"open-cluster-management.io/registration/pkg/hub/managedcluster"
	"open-cluster-management.io/registration/pkg/hub/managedclusterset"
	"open-cluster-management.io/registration/pkg/hub/managedclustersetbinding"
	"open-cluster-management.io/registration/pkg/hub/rbacfinalizerdeletion"
	"open-cluster-management.io/registration/pkg/hub/taint"
)

var ResyncInterval = 5 * time.Minute

// TODO(ycyaoxdu): add placement controllers
func InstallRegistrationPlacementControllers(ctx context.Context, kubeConfig *rest.Config) error {
	eventRecorder := events.NewInMemoryRecorder("registration-controller")

	controllerContext := &controllercmd.ControllerContext{
		KubeConfig:        kubeConfig,
		EventRecorder:     eventRecorder,
		OperatorNamespace: confighub.HubNamespace,
	}

	kubeClient, err := kubernetes.NewForConfig(kubeConfig)
	if err != nil {
		return err
	}

	clusterClient, err := clusterv1client.NewForConfig(kubeConfig)
	if err != nil {
		return err
	}

	workClient, err := workv1client.NewForConfig(kubeConfig)
	if err != nil {
		return err
	}

	addOnClient, err := addonclient.NewForConfig(kubeConfig)
	if err != nil {
		return err
	}

	clusterInformers := clusterv1informers.NewSharedInformerFactory(clusterClient, 10*time.Minute)
	workInformers := workv1informers.NewSharedInformerFactory(workClient, 10*time.Minute)
	kubeInfomers := kubeinformers.NewSharedInformerFactory(kubeClient, 10*time.Minute)
	addOnInformers := addoninformers.NewSharedInformerFactory(addOnClient, 10*time.Minute)

	managedClusterController := managedcluster.NewManagedClusterController(
		kubeClient,
		clusterClient,
		clusterInformers.Cluster().V1().ManagedClusters(),
		controllerContext.EventRecorder,
	)

	taintController := taint.NewTaintController(
		clusterClient,
		clusterInformers.Cluster().V1().ManagedClusters(),
		controllerContext.EventRecorder,
	)

	var csrController factory.Controller
	if features.DefaultHubMutableFeatureGate.Enabled(ocmfeature.V1beta1CSRAPICompatibility) {
		v1CSRSupported, v1beta1CSRSupported, err := helpers.IsCSRSupported(kubeClient)
		if err != nil {
			return errors.Wrapf(err, "failed CSR api discovery")
		}

		if !v1CSRSupported && v1beta1CSRSupported {
			csrController = csr.NewV1beta1CSRApprovingController(
				kubeClient,
				kubeInfomers.Certificates().V1beta1().CertificateSigningRequests(),
				controllerContext.EventRecorder,
			)
			klog.Info("Using v1beta1 CSR api to manage spoke client certificate")
		}
	}
	if csrController == nil {
		csrController = csr.NewCSRApprovingController(
			kubeClient,
			kubeInfomers.Certificates().V1().CertificateSigningRequests(),
			controllerContext.EventRecorder,
		)
	}

	leaseController := lease.NewClusterLeaseController(
		kubeClient,
		clusterClient,
		clusterInformers.Cluster().V1().ManagedClusters(),
		kubeInfomers.Coordination().V1().Leases(),
		ResyncInterval, //TODO: this interval time should be allowed to change from outside
		controllerContext.EventRecorder,
	)

	rbacFinalizerController := rbacfinalizerdeletion.NewFinalizeController(
		kubeInfomers.Rbac().V1().Roles(),
		kubeInfomers.Rbac().V1().RoleBindings(),
		kubeInfomers.Core().V1().Namespaces().Lister(),
		clusterInformers.Cluster().V1().ManagedClusters().Lister(),
		workInformers.Work().V1().ManifestWorks().Lister(),
		kubeClient.RbacV1(),
		controllerContext.EventRecorder,
	)

	managedClusterSetController := managedclusterset.NewManagedClusterSetController(
		clusterClient,
		clusterInformers.Cluster().V1().ManagedClusters(),
		clusterInformers.Cluster().V1beta1().ManagedClusterSets(),
		controllerContext.EventRecorder,
	)

	managedClusterSetBindingController := managedclustersetbinding.NewManagedClusterSetBindingController(
		clusterClient,
		clusterInformers.Cluster().V1beta1().ManagedClusterSets(),
		clusterInformers.Cluster().V1beta1().ManagedClusterSetBindings(),
		controllerContext.EventRecorder,
	)

	clusterroleController := clusterrole.NewManagedClusterClusterroleController(
		kubeClient,
		clusterInformers.Cluster().V1().ManagedClusters(),
		kubeInfomers.Rbac().V1().ClusterRoles(),
		controllerContext.EventRecorder,
	)

	addOnHealthCheckController := addon.NewManagedClusterAddOnHealthCheckController(
		addOnClient,
		addOnInformers.Addon().V1alpha1().ManagedClusterAddOns(),
		clusterInformers.Cluster().V1().ManagedClusters(),
		controllerContext.EventRecorder,
	)

	addOnFeatureDiscoveryController := addon.NewAddOnFeatureDiscoveryController(
		clusterClient,
		clusterInformers.Cluster().V1().ManagedClusters(),
		addOnInformers.Addon().V1alpha1().ManagedClusterAddOns(),
		controllerContext.EventRecorder,
	)

	var defaultManagedClusterSetController, globalManagedClusterSetController factory.Controller
	if features.DefaultHubMutableFeatureGate.Enabled(ocmfeature.DefaultClusterSet) {
		defaultManagedClusterSetController = managedclusterset.NewDefaultManagedClusterSetController(
			clusterClient.ClusterV1beta1(),
			clusterInformers.Cluster().V1beta1().ManagedClusterSets(),
			controllerContext.EventRecorder,
		)
		globalManagedClusterSetController = managedclusterset.NewGlobalManagedClusterSetController(
			clusterClient.ClusterV1beta1(),
			clusterInformers.Cluster().V1beta1().ManagedClusterSets(),
			controllerContext.EventRecorder,
		)
	}

	broadcaster := kubeevents.NewBroadcaster(&kubeevents.EventSinkImpl{Interface: kubeClient.EventsV1()})

	broadcaster.StartRecordingToSink(ctx.Done())

	recorder := broadcaster.NewRecorder(clusterscheme.Scheme, "placementController")

	scheduler := scheduling.NewPluginScheduler(
		scheduling.NewSchedulerHandler(
			clusterClient,
			clusterInformers.Cluster().V1beta1().PlacementDecisions().Lister(),
			clusterInformers.Cluster().V1alpha1().AddOnPlacementScores().Lister(),
			clusterInformers.Cluster().V1().ManagedClusters().Lister(),
			recorder),
	)

	if controllerContext.Server != nil {
		debug := debugger.NewDebugger(
			scheduler,
			clusterInformers.Cluster().V1beta1().Placements(),
			clusterInformers.Cluster().V1().ManagedClusters(),
		)

		controllerContext.Server.Handler.NonGoRestfulMux.HandlePrefix(debugger.DebugPath,
			http.HandlerFunc(debug.Handler))
	}

	schedulingController := scheduling.NewSchedulingController(
		clusterClient,
		clusterInformers.Cluster().V1().ManagedClusters(),
		clusterInformers.Cluster().V1beta1().ManagedClusterSets(),
		clusterInformers.Cluster().V1beta1().ManagedClusterSetBindings(),
		clusterInformers.Cluster().V1beta1().Placements(),
		clusterInformers.Cluster().V1beta1().PlacementDecisions(),
		scheduler,
		controllerContext.EventRecorder, recorder,
	)

	schedulingControllerResync := scheduling.NewSchedulingControllerResync(
		clusterClient,
		clusterInformers.Cluster().V1().ManagedClusters(),
		clusterInformers.Cluster().V1beta1().ManagedClusterSets(),
		clusterInformers.Cluster().V1beta1().ManagedClusterSetBindings(),
		clusterInformers.Cluster().V1beta1().Placements(),
		clusterInformers.Cluster().V1beta1().PlacementDecisions(),
		scheduler,
		controllerContext.EventRecorder, recorder,
	)
	go clusterInformers.Start(ctx.Done())
	go workInformers.Start(ctx.Done())
	go kubeInfomers.Start(ctx.Done())
	go addOnInformers.Start(ctx.Done())

	go managedClusterController.Run(ctx, 1)
	go taintController.Run(ctx, 1)
	go csrController.Run(ctx, 1)
	go leaseController.Run(ctx, 1)
	go rbacFinalizerController.Run(ctx, 1)
	go managedClusterSetController.Run(ctx, 1)
	go managedClusterSetBindingController.Run(ctx, 1)
	go clusterroleController.Run(ctx, 1)
	go addOnHealthCheckController.Run(ctx, 1)
	go addOnFeatureDiscoveryController.Run(ctx, 1)
	if features.DefaultHubMutableFeatureGate.Enabled(ocmfeature.DefaultClusterSet) {
		go defaultManagedClusterSetController.Run(ctx, 1)
		go globalManagedClusterSetController.Run(ctx, 1)
	}
	go schedulingController.Run(ctx, 1)
	go schedulingControllerResync.Run(ctx, 1)

	<-ctx.Done()
	return nil
}
