// Copyright 2016 The prometheus-operator Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package elasticsearch

import (
	"bytes"
	"fmt"
	"reflect"
	"strings"
	"time"

	"github.com/galexrt/elasticsearch-operator/pkg/analytics"
	"github.com/galexrt/elasticsearch-operator/pkg/client/monitoring/v1alpha1"
	"github.com/galexrt/elasticsearch-operator/pkg/k8sutil"

	"github.com/galexrt/elasticsearch-operator/pkg/config"
	"github.com/galexrt/elasticsearch-operator/third_party/workqueue"
	"github.com/go-kit/kit/log"
	"github.com/pkg/errors"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/labels"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/pkg/api"
	"k8s.io/client-go/pkg/api/v1"
	"k8s.io/client-go/pkg/apis/apps/v1beta1"
	extensionsobj "k8s.io/client-go/pkg/apis/extensions/v1beta1"
	"k8s.io/client-go/tools/cache"
)

const (
	tprElasticsearch = "elasticsearch." + v1alpha1.TPRGroup
	configFilename   = "elasticsearch.yml"
	log4jFilename    = "log4j2.properties"
	jvmOpts          = "jvm.options"

	resyncPeriod = 5 * time.Minute
)

// Operator manages life cycle of Elasticsearch deployments and
// monitoring configurations.
type Operator struct {
	kclient *kubernetes.Clientset
	mclient *v1alpha1.MonitoringV1alpha1Client
	logger  log.Logger

	elastInf cache.SharedIndexInformer
	secrInf  cache.SharedIndexInformer
	ssetInf  cache.SharedIndexInformer

	queue workqueue.RateLimitingInterface

	host   string
	config config.Config
}

// New creates a new controller.
func New(conf config.Config, logger log.Logger) (*Operator, error) {
	cfg, err := k8sutil.NewClusterConfig(conf.Host, conf.TLSInsecure, &conf.TLSConfig)
	if err != nil {
		return nil, err
	}
	client, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		return nil, err
	}

	mclient, err := v1alpha1.NewForConfig(cfg)
	if err != nil {
		return nil, err
	}

	c := &Operator{
		kclient: client,
		mclient: mclient,
		logger:  logger,
		queue:   workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), "elasticsearch"),
		host:    cfg.Host,
		config:  conf,
	}

	c.elastInf = cache.NewSharedIndexInformer(
		&cache.ListWatch{
			ListFunc:  mclient.Elastichearchs(api.NamespaceAll).List,
			WatchFunc: mclient.Elastichearchs(api.NamespaceAll).Watch,
		},
		&v1alpha1.Elasticsearch{}, resyncPeriod, cache.Indexers{},
	)
	c.elastInf.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    c.handleAddElasticsearch,
		DeleteFunc: c.handleDeleteElasticsearch,
		UpdateFunc: c.handleUpdateElasticsearch,
	})

	c.secrInf = cache.NewSharedIndexInformer(
		cache.NewListWatchFromClient(c.kclient.Core().RESTClient(), "secrets", api.NamespaceAll, nil),
		&v1.Secret{}, resyncPeriod, cache.Indexers{},
	)
	c.secrInf.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    c.handleSecretAdd,
		DeleteFunc: c.handleSecretDelete,
		UpdateFunc: c.handleSecretUpdate,
	})

	c.ssetInf = cache.NewSharedIndexInformer(
		cache.NewListWatchFromClient(c.kclient.Apps().RESTClient(), "statefulsets", api.NamespaceAll, nil),
		&v1beta1.StatefulSet{}, resyncPeriod, cache.Indexers{},
	)
	c.ssetInf.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    c.handleAddStatefulSet,
		DeleteFunc: c.handleDeleteStatefulSet,
		UpdateFunc: c.handleUpdateStatefulSet,
	})

	return c, nil
}

// Run the controller.
func (c *Operator) Run(stopc <-chan struct{}) error {
	defer c.queue.ShutDown()

	errChan := make(chan error)
	go func() {
		v, err := c.kclient.Discovery().ServerVersion()
		if err != nil {
			errChan <- errors.Wrap(err, "communicating with server failed")
			return
		}
		c.logger.Log("msg", "connection established", "cluster-version", v)

		if err := c.createTPRs(); err != nil {
			errChan <- errors.Wrap(err, "creating TPRs failed")
			return
		}
		errChan <- nil
	}()

	select {
	case err := <-errChan:
		if err != nil {
			return err
		}
		c.logger.Log("msg", "TPR API endpoints ready")
	case <-stopc:
		return nil
	}

	go c.worker()

	go c.elastInf.Run(stopc)
	go c.secrInf.Run(stopc)
	go c.ssetInf.Run(stopc)

	<-stopc
	return nil
}

func (c *Operator) keyFunc(obj interface{}) (string, bool) {
	k, err := cache.DeletionHandlingMetaNamespaceKeyFunc(obj)
	if err != nil {
		c.logger.Log("msg", "creating key failed", "err", err)
		return k, false
	}
	return k, true
}

func (c *Operator) handleAddElasticsearch(obj interface{}) {
	key, ok := c.keyFunc(obj)
	if !ok {
		return
	}

	analytics.ElasticsearchCreated()
	c.logger.Log("msg", "Elasticsearch added", "key", key)
	c.enqueue(key)
}

func (c *Operator) handleDeleteElasticsearch(obj interface{}) {
	key, ok := c.keyFunc(obj)
	if !ok {
		return
	}

	analytics.ElasticsearchDeleted()
	c.logger.Log("msg", "Elasticsearch deleted", "key", key)
	c.enqueue(key)
}

func (c *Operator) handleUpdateElasticsearch(old, cur interface{}) {
	key, ok := c.keyFunc(cur)
	if !ok {
		return
	}

	c.logger.Log("msg", "Elasticsearch updated", "key", key)
	c.enqueue(key)
}

func (c *Operator) handleSecretDelete(obj interface{}) {
	o, ok := c.getObject(obj)
	if ok {
		c.enqueueForNamespace(o.GetNamespace())
	}
}

func (c *Operator) handleSecretUpdate(old, cur interface{}) {
	o, ok := c.getObject(cur)
	if ok {
		c.enqueueForNamespace(o.GetNamespace())
	}
}

func (c *Operator) handleSecretAdd(obj interface{}) {
	o, ok := c.getObject(obj)
	if ok {
		c.enqueueForNamespace(o.GetNamespace())
	}
}

func (c *Operator) getObject(obj interface{}) (metav1.Object, bool) {
	ts, ok := obj.(cache.DeletedFinalStateUnknown)
	if ok {
		obj = ts.Obj
	}

	o, err := meta.Accessor(obj)
	if err != nil {
		c.logger.Log("msg", "get object failed", "err", err)
		return nil, false
	}
	return o, true
}

// enqueue adds a key to the queue. If obj is a key already it gets added directly.
// Otherwise, the key is extracted via keyFunc.
func (c *Operator) enqueue(obj interface{}) {
	if obj == nil {
		return
	}

	key, ok := obj.(string)
	if !ok {
		key, ok = c.keyFunc(obj)
		if !ok {
			return
		}
	}

	c.queue.Add(key)
}

// enqueueForNamespace enqueues all Elasticsearch object keys that belong to the given namespace.
func (c *Operator) enqueueForNamespace(ns string) {
	cache.ListAll(c.elastInf.GetStore(), labels.Everything(), func(obj interface{}) {
		p := obj.(*v1alpha1.Elasticsearch)
		if p.Namespace == ns {
			c.enqueue(p)
		}
	})
}

// worker runs a worker thread that just dequeues items, processes them, and marks them done.
// It enforces that the syncHandler is never invoked concurrently with the same key.
func (c *Operator) worker() {
	for c.processNextWorkItem() {
	}
}

func (c *Operator) processNextWorkItem() bool {
	key, quit := c.queue.Get()
	if quit {
		return false
	}
	defer c.queue.Done(key)

	err := c.sync(key.(string))
	if err == nil {
		c.queue.Forget(key)
		return true
	}

	utilruntime.HandleError(errors.Wrap(err, fmt.Sprintf("Sync %q failed", key)))
	c.queue.AddRateLimited(key)

	return true
}

func (c *Operator) elasticsearchForStatefulSet(sset interface{}) *v1alpha1.Elasticsearch {
	key, ok := c.keyFunc(sset)
	if !ok {
		return nil
	}

	elasKey := statefulSetKeyToElasticsearchKey(key)
	p, exists, err := c.elastInf.GetStore().GetByKey(elasKey)
	if err != nil {
		c.logger.Log("msg", "Elasticsearch lookup failed", "err", err)
		return nil
	}
	if !exists {
		return nil
	}
	return p.(*v1alpha1.Elasticsearch)
}

func elasticsearchNameFromStatefulSetName(name string) string {
	return strings.Split(strings.TrimPrefix(name, "elasticsearch-"), "-")[0]
}

func statefulSetNameFromElasticsearchName(name string) string {
	return "elasticsearch-" + name
}

func statefulSetKeyToElasticsearchKey(key string) string {
	keyParts := strings.Split(key, "/")
	return keyParts[0] + "/" + strings.TrimPrefix(keyParts[1], "elasticsearch-")
}

func elasticsearchKeyToStatefulSetKey(key string) string {
	keyParts := strings.Split(key, "/")
	return keyParts[0] + "/elasticsearch-" + keyParts[1]
}

func (c *Operator) handleDeleteStatefulSet(obj interface{}) {
	if ps := c.elasticsearchForStatefulSet(obj); ps != nil {
		c.enqueue(ps)
	}
}

func (c *Operator) handleAddStatefulSet(obj interface{}) {
	if ps := c.elasticsearchForStatefulSet(obj); ps != nil {
		c.enqueue(ps)
	}
}

func (c *Operator) handleUpdateStatefulSet(oldo, curo interface{}) {
	old := oldo.(*v1beta1.StatefulSet)
	cur := curo.(*v1beta1.StatefulSet)

	c.logger.Log("msg", "update handler", "old", old.ResourceVersion, "cur", cur.ResourceVersion)

	// Periodic resync may resend the deployment without changes in-between.
	// Also breaks loops created by updating the resource ourselves.
	if old.ResourceVersion == cur.ResourceVersion {
		return
	}

	if ps := c.elasticsearchForStatefulSet(cur); ps != nil {
		c.enqueue(ps)
	}
}

func (c *Operator) sync(key string) error {
	obj, exists, err := c.elastInf.GetIndexer().GetByKey(key)
	if err != nil {
		return err
	}
	if !exists {
		// TODO(fabxc): we want to do server side deletion due to the variety of
		// resources we create.
		// Doing so just based on the deletion event is not reliable, so
		// we have to garbage collect the controller-created resources in some other way.
		//
		// Let's rely on the index key matching that of the created configmap and StatefulSet for now.
		// This does not work if we delete Elasticsearch resources as the
		// controller is not running – that could be solved via garbage collection later.
		return c.destroyElasticsearch(key)
	}

	el := obj.(*v1alpha1.Elasticsearch)
	if el.Spec.Paused {
		return nil
	}

	c.logger.Log("msg", "sync elasticsearch", "key", key)

	if err := c.createConfig(el); err != nil {
		return errors.Wrap(err, "creating config failed")
	}
	for _, tkey := range []string{
		"master",
		"data",
		"ingest",
	} {
		// Create Secret if it doesn't exist.
		s, err := makeEmptyConfig(el.ObjectMeta.Name + "-" + tkey)
		if err != nil {
			return errors.Wrap(err, "generating empty config secret failed")
		}
		if _, err := c.kclient.Core().Secrets(el.Namespace).Create(s); err != nil && !apierrors.IsAlreadyExists(err) {
			return errors.Wrap(err, "creating empty config file failed")
		}
	}

	// Create governing service if it doesn't exist.
	svcClient := c.kclient.Core().Services(el.Namespace)
	if err := k8sutil.CreateOrUpdateService(svcClient, makeGovenorningService(el)); err != nil {
		return errors.Wrap(err, "synchronizing governing service failed")
	}

	getOptions := metav1.GetOptions{}

	for _, tkey := range []string{
		"master",
		"data",
		"ingest",
		"discovery",
	} {
		// Create statefulset services if they don't exist.
		_, err = svcClient.Get(prefixedName(el.Name)+"-"+tkey, getOptions)
		if k8sutil.IsResourceNotFoundError(err) {
			if _, err = svcClient.Create(makeStatefulSetService(prefixedName(el.Name)+"-"+tkey, tkey)); err != nil {
				return errors.Wrap(err, "synchronizing statefulset service failed")
			}
		}
	}

	ssetClient := c.kclient.Apps().StatefulSets(el.Namespace)

	ok := false

	for tkey, p := range map[string]*v1alpha1.ElasticsearchPartSpec{
		"master": el.Spec.Master,
		"data":   el.Spec.Data,
		"ingest": el.Spec.Ingest,
	} {
		// Ensure we have a StatefulSet running Elasticsearch master, data and ingest deployed.
		obj, exists, err = c.ssetInf.GetIndexer().GetByKey(elasticsearchKeyToStatefulSetKey(key + "-" + tkey))
		if err != nil {
			return errors.Wrap(err, "retrieving statefulset failed")
		}
		if !exists {
			sset, err := makeStatefulSets(*el, tkey, p, nil, &c.config)
			if err != nil {
				return errors.Wrap(err, "creating statefulsets failed")
			}
			if _, err := ssetClient.Create(sset); err != nil {
				return errors.Wrap(err, "creating statefulsets failed")
			}
			ok = true
			continue
		}
		sset, err := makeStatefulSets(*el, tkey, p, obj.(*v1beta1.StatefulSet), &c.config)
		if err != nil {
			return errors.Wrap(err, "updating statefulsets failed")
		}
		if _, err := ssetClient.Update(sset); err != nil {
			return errors.Wrap(err, "updating statefulset failed")
		}
	}
	if ok {
		return nil
	}

	err = c.syncVersion(key, el)
	if err != nil {
		return errors.Wrap(err, "syncing version failed")
	}

	return nil
}

func ListOptions(name string) metav1.ListOptions {
	return metav1.ListOptions{
		LabelSelector: fields.SelectorFromSet(fields.Set(map[string]string{
			"app":           "elasticsearch",
			"elasticsearch": name,
		})).String(),
	}
}

// syncVersion ensures that all running pods for a Elasticsearch have the required version.
// It kills pods with the wrong version one-after-one and lets the StatefulSet controller
// create new pods.
//
// TODO(fabxc): remove this once the StatefulSet controller learns how to do rolling updates.
func (c *Operator) syncVersion(key string, el *v1alpha1.Elasticsearch) error {
	statuses, oldPods, err := ElasticsearchStatus(c.kclient, el)
	if err != nil {
		return errors.Wrap(err, "retrieving Elasticsearch status failed")
	}

	for tkey, p := range map[string]*v1alpha1.ElasticsearchPartSpec{
		"master": el.Spec.Master,
		"data":   el.Spec.Data,
		"ingest": el.Spec.Ingest,
	} {
		status := statuses[tkey]
		// If the StatefulSet is still busy scaling, don't interfere by killing pods.
		// We enqueue ourselves again to until the StatefulSet is ready.
		expectedReplicas := int32(1)
		if p.Replicas != nil {
			expectedReplicas = *p.Replicas
		}
		if status.Replicas != expectedReplicas {
			return fmt.Errorf("scaling in progress, %d expected replicas, %d found replicas", expectedReplicas, status.Replicas)
		}
		if status.Replicas == 0 {
			return nil
		}
		if len(oldPods[tkey]) == 0 {
			return nil
		}
		if status.UnavailableReplicas > 0 {
			return fmt.Errorf("waiting for %d unavailable pods to become ready", status.UnavailableReplicas)
		}

		// TODO(fabxc): delete oldest pod first.
		if err := c.kclient.Core().Pods(el.Namespace).Delete(oldPods[tkey][0].Name, nil); err != nil {
			return err
		}
		// If there are further pods that need updating, we enqueue ourselves again.
		if len(oldPods) > 1 {
			return fmt.Errorf("%d out-of-date pods remaining", len(oldPods)-1)
		}
	}
	return nil
}

// ElasticsearchStatus evaluates the current status of a Elasticsearch deployment with respect
// to its specified resource object. It return the status and a list of pods that
// are not updated.
func ElasticsearchStatus(kclient kubernetes.Interface, el *v1alpha1.Elasticsearch) (map[string]*v1alpha1.ElasticsearchStatus, map[string][]v1.Pod, error) {
	ress := map[string]*v1alpha1.ElasticsearchStatus{}

	var oldPods map[string][]v1.Pod

	for tkey, _ := range map[string]*v1alpha1.ElasticsearchPartSpec{
		"master": el.Spec.Master,
		"data":   el.Spec.Data,
		"ingest": el.Spec.Ingest,
	} {
		res := &v1alpha1.ElasticsearchStatus{
			Paused:   el.Spec.Paused,
			Replicas: int32(0),
		}
		name := el.ObjectMeta.Name + "-" + tkey

		pods, err := kclient.Core().Pods(el.Namespace).List(ListOptions(name))
		if err != nil {
			return nil, nil, errors.Wrap(err, "retrieving pods of failed")
		}
		sset, err := kclient.Apps().StatefulSets(el.Namespace).Get(statefulSetNameFromElasticsearchName(name), metav1.GetOptions{})
		if err != nil {
			return nil, nil, errors.Wrap(err, "retrieving stateful set failed")
		}

		res.Replicas += int32(len(pods.Items))

		for _, pod := range pods.Items {
			ready, err := k8sutil.PodRunningAndReady(pod)
			if err != nil {
				return nil, nil, errors.Wrap(err, "cannot determine pod ready state")
			}
			if ready {
				res.AvailableReplicas++
				// TODO(fabxc): detect other fields of the pod template that are mutable.
				if needsUpdate(&pod, sset.Spec.Template) {
					oldPods[tkey] = append(oldPods[tkey], pod)
				} else {
					res.UpdatedReplicas++
				}
				continue
			}
			res.UnavailableReplicas++
		}
		ress[tkey] = res
	}
	return ress, oldPods, nil
}

// needsUpdate checks whether the given pod conforms with the pod template spec
// for various attributes that are influenced by the Elasticsearch TPR settings.
func needsUpdate(pod *v1.Pod, tmpl v1.PodTemplateSpec) bool {
	c1 := pod.Spec.Containers[0]
	c2 := tmpl.Spec.Containers[0]

	if c1.Image != c2.Image {
		return true
	}
	if !reflect.DeepEqual(c1.Resources, c2.Resources) {
		return true
	}
	if !reflect.DeepEqual(c1.Args, c2.Args) {
		return true
	}
	if !reflect.DeepEqual(c1.Env, c2.Env) {
		return true
	}

	return false
}

func (c *Operator) destroyElasticsearch(key string) error {
	var sset *v1beta1.StatefulSet

	for _, tkey := range []string{
		"master",
		"data",
		"ingest",
	} {
		ssetKey := elasticsearchKeyToStatefulSetKey(key + "-" + tkey)
		obj, exists, err := c.ssetInf.GetStore().GetByKey(ssetKey)
		if err != nil {
			return errors.Wrap(err, "retrieving statefulset from cache failed")
		}
		if !exists {
			continue
		}
		sset = obj.(*v1beta1.StatefulSet)
		*sset.Spec.Replicas = 0

		// Update the replica count to 0 and wait for all pods to be deleted.
		ssetClient := c.kclient.Apps().StatefulSets(sset.Namespace)

		if _, err := ssetClient.Update(sset); err != nil {
			return errors.Wrap(err, "updating statefulset for scale-down failed")
		}

		// StatefulSet can be deleted.
		if err := ssetClient.Delete(sset.Name, nil); err != nil {
			return errors.Wrap(err, "deleting statefulset failed")
		}
	}

	nameParts := strings.Split(key, "/")
	name := nameParts[1]
	namespace := nameParts[0]

	podClient := c.kclient.Core().Pods(namespace)

	// TODO(fabxc): temprorary solution until StatefulSet status provides necessary info to know
	// whether scale-down completed.
	for {
		pods, err := podClient.List(ListOptions(elasticsearchNameFromStatefulSetName(name)))
		if err != nil {
			return errors.Wrap(err, "retrieving pods of statefulset failed")
		}

		if len(pods.Items) == 0 {
			break
		}
		time.Sleep(100 * time.Millisecond)
		if err := c.kclient.Core().Pods(namespace).Delete(pods.Items[0].Name, nil); err != nil {
			if !k8sutil.IsResourceNotFoundError(err) {
				return err
			}
			continue
		}
		c.kclient.Core().Pods(namespace).Delete(pods.Items[0].Name, nil)
	}

	for _, tkey := range []string{
		"master",
		"data",
		"ingest",
	} {
		// Delete the auto-generate configuration.
		s := c.kclient.Core().Secrets(namespace)
		secret, err := s.Get(configSecretName(name+"-"+tkey), metav1.GetOptions{})
		if err != nil && !apierrors.IsNotFound(err) {
			return errors.Wrap(err, "retrieving config Secret failed")
		}
		if apierrors.IsNotFound(err) {
			// Secret does not exist so nothing to clean up
			continue
		}

		value, found := secret.Labels[managedByOperatorLabel]
		if found && value == managedByOperatorLabelValue {
			if err := s.Delete(configSecretName(name+"-"+tkey), nil); err != nil {
				return errors.Wrap(err, "deleting config Secret failed")
			}
		}
	}

	for _, tkey := range []string{
		"master",
		"data",
		"ingest",
		"discovery",
	} {
		// Delete statefulset services
		svcClient := c.kclient.Core().Services(namespace)
		err := svcClient.Delete(prefixedName(name)+"-"+tkey, &metav1.DeleteOptions{})
		if err != nil && !k8sutil.IsResourceNotFoundError(err) {
			return errors.Wrap(err, "synchronizing statefulset service failed")
		}
	}

	return nil
}

func (c *Operator) createConfig(p *v1alpha1.Elasticsearch) error {
	var err error
	sClient := c.kclient.CoreV1().Secrets(p.Namespace)
	for _, tkey := range []string{
		"master",
		"data",
		"ingest",
	} {
		// Update secret based on the most recent configuration.
		config, err := generateConfig(p, tkey)
		if err != nil {
			return errors.Wrap(err, "generating config failed")
		}

		s, err := makeConfigSecret(p.Name + "-" + tkey)
		if err != nil {
			return errors.Wrap(err, "generating base secret failed")
		}
		s.ObjectMeta.Annotations = map[string]string{
			"generated": "true",
		}
		s.Data = config

		curSecret, err := sClient.Get(s.Name, metav1.GetOptions{})
		if apierrors.IsNotFound(err) {
			c.logger.Log("msg", "creating configuration")
			_, err = sClient.Create(s)
			if err != nil {
				return err
			}
			continue
		}

		ok := -1

		for configFilename, generatedConf := range s.Data {
			curConfig, curConfigFound := curSecret.Data[configFilename]
			if curConfigFound {
				if bytes.Equal(curConfig, generatedConf) {
					c.logger.Log("msg", "updating config skipped, no configuration change")
					if ok == -1 {
						ok = 1
					}
				} else {
					ok = 0
					c.logger.Log("msg", "current config has changed")
				}
			} else {
				ok = 0
				c.logger.Log("msg", "no current config found", "currentConfigFound", curConfigFound)
			}
		}
		if ok == 1 {
			continue
		}

		c.logger.Log("msg", "updating configuration")
		_, err = sClient.Update(s)
		if err != nil {
			return err
		}
	}
	return err
}

func (c *Operator) createTPRs() error {
	tprs := []*extensionsobj.ThirdPartyResource{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: tprElasticsearch,
			},
			Versions: []extensionsobj.APIVersion{
				{Name: v1alpha1.TPRVersion},
			},
			Description: "Managed Elasticsearch instance(s)",
		},
	}
	tprClient := c.kclient.Extensions().ThirdPartyResources()

	for _, tpr := range tprs {
		if _, err := tprClient.Create(tpr); err != nil && !apierrors.IsAlreadyExists(err) {
			return err
		}
		c.logger.Log("msg", "TPR created", "tpr", tpr.Name)
	}

	// We have to wait for the TPRs to be ready. Otherwise the initial watch may fail.
	return k8sutil.WaitForTPRReady(c.kclient.CoreV1().RESTClient(), v1alpha1.TPRGroup, v1alpha1.TPRVersion, v1alpha1.TPRElasticsearchName)
}
