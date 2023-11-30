// Code generated by client-gen. DO NOT EDIT.

package v1

import (
	"context"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	types "k8s.io/apimachinery/pkg/types"
	watch "k8s.io/apimachinery/pkg/watch"
	rest "k8s.io/client-go/rest"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	scheme "open-cluster-management.io/api/client/operator/clientset/versioned/scheme"
	v1 "open-cluster-management.io/api/operator/v1"
	"github.com/spf13/pflag"
)

// KlusterletsGetter has a method to return a KlusterletInterface.
// A group's client should implement this interface.
type KlusterletsGetter interface {
	Klusterlets() KlusterletInterface
}

// KlusterletInterface has methods to work with Klusterlet resources.
type KlusterletInterface interface {
	Create(ctx context.Context, klusterlet *v1.Klusterlet, opts metav1.CreateOptions) (*v1.Klusterlet, error)
	Update(ctx context.Context, klusterlet *v1.Klusterlet, opts metav1.UpdateOptions) (*v1.Klusterlet, error)
	UpdateStatus(ctx context.Context, klusterlet *v1.Klusterlet, opts metav1.UpdateOptions) (*v1.Klusterlet, error)
	Delete(ctx context.Context, name string, opts metav1.DeleteOptions) error
	DeleteCollection(ctx context.Context, opts metav1.DeleteOptions, listOpts metav1.ListOptions) error
	Get(ctx context.Context, name string, opts metav1.GetOptions) (*v1.Klusterlet, error)
	List(ctx context.Context, opts metav1.ListOptions) (*v1.KlusterletList, error)
	Watch(ctx context.Context, opts metav1.ListOptions) (watch.Interface, error)
	Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts metav1.PatchOptions, subresources ...string) (result *v1.Klusterlet, err error)
	KlusterletExpansion
}

// klusterlets implements KlusterletInterface
type klusterlets struct {
	client rest.Interface
}

// newKlusterlets returns a Klusterlets
func newKlusterlets(c *OperatorV1Client) *klusterlets {
	return &klusterlets{
		client: c.RESTClient(),
	}
}

// Get takes name of the klusterlet, and returns the corresponding klusterlet object, and an error if there is any.
func (c *klusterlets) Get(ctx context.Context, name string, options metav1.GetOptions) (result *v1.Klusterlet, err error) {
	result = &v1.Klusterlet{}
	err = c.client.Get().
		Resource("klusterlets").
		Name(name).
		VersionedParams(&options, scheme.ParameterCodec).
		Do(ctx).
		Into(result)
	return
}

// List takes label and field selectors, and returns the list of Klusterlets that match those selectors.
func (c *klusterlets) List(ctx context.Context, opts metav1.ListOptions) (result *v1.KlusterletList, err error) {
	var timeout time.Duration
	if opts.TimeoutSeconds != nil {
		timeout = time.Duration(*opts.TimeoutSeconds) * time.Second
	}
	result = &v1.KlusterletList{}
	err = c.client.Get().
		Resource("klusterlets").
		VersionedParams(&opts, scheme.ParameterCodec).
		Timeout(timeout).
		Do(ctx).
		Into(result)
	return
}

// Watch returns a watch.Interface that watches the requested klusterlets.
func (c *klusterlets) Watch(ctx context.Context, opts metav1.ListOptions) (watch.Interface, error) {
	var timeout time.Duration
	if opts.TimeoutSeconds != nil {
		timeout = time.Duration(*opts.TimeoutSeconds) * time.Second
	}
	opts.Watch = true
	return c.client.Get().
		Resource("klusterlets").
		VersionedParams(&opts, scheme.ParameterCodec).
		Timeout(timeout).
		Watch(ctx)
}

// Create takes the representation of a klusterlet and creates it.  Returns the server's representation of the klusterlet, and an error, if there is any.
func (c *klusterlets) Create(ctx context.Context, klusterlet *v1.Klusterlet, opts metav1.CreateOptions) (result *v1.Klusterlet, err error) {
	result = &v1.Klusterlet{}
	err = c.client.Post().
		Resource("klusterlets").
		VersionedParams(&opts, scheme.ParameterCodec).
		Body(klusterlet).
		Do(ctx).
		Into(result)
	return
}

// Update takes the representation of a klusterlet and updates it. Returns the server's representation of the klusterlet, and an error, if there is any.
func (c *klusterlets) Update(ctx context.Context, klusterlet *v1.Klusterlet, opts metav1.UpdateOptions) (result *v1.Klusterlet, err error) {
	result = &v1.Klusterlet{}
	err = c.client.Put().
		Resource("klusterlets").
		Name(klusterlet.Name).
		VersionedParams(&opts, scheme.ParameterCodec).
		Body(klusterlet).
		Do(ctx).
		Into(result)
	return
}

// UpdateStatus was generated because the type contains a Status member.
// Add a +genclient:noStatus comment above the type to avoid generating UpdateStatus().
func (c *klusterlets) UpdateStatus(ctx context.Context, klusterlet *v1.Klusterlet, opts metav1.UpdateOptions) (result *v1.Klusterlet, err error) {
	result = &v1.Klusterlet{}
	err = c.client.Put().
		Resource("klusterlets").
		Name(klusterlet.Name).
		SubResource("status").
		VersionedParams(&opts, scheme.ParameterCodec).
		Body(klusterlet).
		Do(ctx).
		Into(result)
	return
}

// Delete takes name of the klusterlet and deletes it. Returns an error if one occurs.
func (c *klusterlets) Delete(ctx context.Context, name string, opts metav1.DeleteOptions) error {
	return c.client.Delete().
		Resource("klusterlets").
		Name(name).
		Body(&opts).
		Do(ctx).
		Error()
}

// DeleteCollection deletes a collection of objects.
func (c *klusterlets) DeleteCollection(ctx context.Context, opts metav1.DeleteOptions, listOpts metav1.ListOptions) error {
	var timeout time.Duration
	if listOpts.TimeoutSeconds != nil {
		timeout = time.Duration(*listOpts.TimeoutSeconds) * time.Second
	}
	return c.client.Delete().
		Resource("klusterlets").
		VersionedParams(&listOpts, scheme.ParameterCodec).
		Timeout(timeout).
		Body(&opts).
		Do(ctx).
		Error()
}

// Patch applies the patch and returns the patched klusterlet.
func (c *klusterlets) Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts metav1.PatchOptions, subresources ...string) (result *v1.Klusterlet, err error) {
	result = &v1.Klusterlet{}
	err = c.client.Patch(pt).
		Resource("klusterlets").
		Name(name).
		SubResource(subresources...).
		VersionedParams(&opts, scheme.ParameterCodec).
		Body(data).
		Do(ctx).
		Into(result)
	return
}

type KlusterletFeatureController interface {
	// SetupWithManager sets up the controller with the Manager.
	SetupWithManager(mgr ctrl.Manager) error
	SetClient(c client.Client)
	//SetRecorder(r record.EventRecorder)
}

type KlusterletFeatureRoutine interface {
	Start(ctx context.Context)
	SetClient(c client.Client)
}

type KlusterletFeature struct {

	// the name of KlusterletFeature
	Name string

	reconcilers []reconcile.Reconciler

	routines []KlusterletFeatureRoutine

	fs *pflag.FlagSet

	options ctrl.Options

}


// WithControllerManagerOptions sets the controller manager options.
// problem: the manager is create yet. how to set the options?
func (k *KlusterletFeature) WithControllerManagerOptions(options manager.Options) *KlusterletFeature {
	//TODO
	k.options = options
	return k
}

func (k *KlusterletFeature) WithReconciler(r reconcile.Reconciler) *KlusterletFeature {
	//TODO
	if k.reconcilers == nil {
		k.reconcilers = make([]reconcile.Reconciler, 0)
	}
	k.reconcilers = append(k.reconcilers, r)
	return k
}

func (k *KlusterletFeature) WithRoutine(s KlusterletFeatureRoutine) *KlusterletFeature {
	//TODO
	if k.routines == nil {
		k.routines = make([]KlusterletFeatureRoutine, 0)
	}
	k.routines = append(k.routines, s)
	return k
}

func (k *KlusterletFeature) WithFlags(fs *pflag.FlagSet) *KlusterletFeature {
	k.fs = fs
	return k
}

func (k *KlusterletFeature) Complete(ctx context.Context, mgr ctrl.Manager) (*KlusterletFeature, error) {
	//TODO
	for _, r := range k.reconcilers {
		controller := r.(KlusterletFeatureController)
		controller.SetClient(mgr.GetClient())
		//controller.SetRecorder(mgr.GetEventRecorderFor(k.Name))
		if err := controller.SetupWithManager(mgr); err != nil {
			return nil, err
		}
	}

	for _, s := range k.routines {
		s.SetClient(mgr.GetClient())
		go s.Start(ctx)
	}
	return k, nil
}

func (k *KlusterletFeature) GetOptions() ctrl.Options {
	return k.options
}
