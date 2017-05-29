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

package curator

import (
	"bytes"
	"fmt"
	"reflect"
	"strings"
	"time"

	"github.com/galexrt/elasticsearch-operator/pkg/analytics"
	"github.com/galexrt/elasticsearch-operator/pkg/client/monitoring/v1alpha1"
	"github.com/galexrt/elasticsearch-operator/pkg/config"
	"github.com/galexrt/elasticsearch-operator/pkg/k8sutil"

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
	"k8s.io/client-go/pkg/apis/batch/v2alpha1"
	extensionsobj "k8s.io/client-go/pkg/apis/extensions/v1beta1"
	"k8s.io/client-go/tools/cache"
)

const (
	tprCurator      = "curator." + v1alpha1.TPRGroup
	configFilename  = "curator.yml"
	actionsFilename = "action_file.yml"

	resyncPeriod = 5 * time.Minute
)

// Operator manages lify cycle of Curator deployments and
// monitoring configurations.
type Operator struct {
	kclient *kubernetes.Clientset
	mclient *v1alpha1.MonitoringV1alpha1Client
	logger  log.Logger

	curatInf cache.SharedIndexInformer
	secrInf  cache.SharedIndexInformer
	cjInf    cache.SharedIndexInformer

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
		queue:   workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), "curator"),
		host:    cfg.Host,
		config:  conf,
	}

	c.curatInf = cache.NewSharedIndexInformer(
		&cache.ListWatch{
			ListFunc:  mclient.Curators(api.NamespaceAll).List,
			WatchFunc: mclient.Curators(api.NamespaceAll).Watch,
		},
		&v1alpha1.Curator{}, resyncPeriod, cache.Indexers{},
	)
	c.curatInf.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    c.handleAddCurator,
		DeleteFunc: c.handleDeleteCurator,
		UpdateFunc: c.handleUpdateCurator,
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

	c.cjInf = cache.NewSharedIndexInformer(
		cache.NewListWatchFromClient(c.kclient.BatchV2alpha1().RESTClient(), "cronjobs", api.NamespaceAll, nil),
		&v2alpha1.CronJob{}, resyncPeriod, cache.Indexers{},
	)
	c.cjInf.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    c.handleAddCronJob,
		DeleteFunc: c.handleDeleteCronJob,
		UpdateFunc: c.handleUpdateCronJob,
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

	go c.curatInf.Run(stopc)
	go c.secrInf.Run(stopc)
	go c.cjInf.Run(stopc)

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

func (c *Operator) handleAddCurator(obj interface{}) {
	key, ok := c.keyFunc(obj)
	if !ok {
		return
	}

	analytics.CuratorCreated()
	c.logger.Log("msg", "Curator added", "key", key)
	c.enqueue(key)
}

func (c *Operator) handleDeleteCurator(obj interface{}) {
	key, ok := c.keyFunc(obj)
	if !ok {
		return
	}

	analytics.CuratorDeleted()
	c.logger.Log("msg", "Curator deleted", "key", key)
	c.enqueue(key)
}

func (c *Operator) handleUpdateCurator(old, cur interface{}) {
	key, ok := c.keyFunc(cur)
	if !ok {
		return
	}

	c.logger.Log("msg", "Curator updated", "key", key)
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

// enqueueForNamespace enqueues all Curator object keys that belong to the given namespace.
func (c *Operator) enqueueForNamespace(ns string) {
	cache.ListAll(c.curatInf.GetStore(), labels.Everything(), func(obj interface{}) {
		p := obj.(*v1alpha1.Curator)
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

func (c *Operator) curatorForCronJob(cj interface{}) *v1alpha1.Curator {
	key, ok := c.keyFunc(cj)
	if !ok {
		return nil
	}

	promKey := cronJobKeyToCuratorKey(key)
	p, exists, err := c.curatInf.GetStore().GetByKey(promKey)
	if err != nil {
		c.logger.Log("msg", "Curator lookup failed", "err", err)
		return nil
	}
	if !exists {
		return nil
	}
	return p.(*v1alpha1.Curator)
}

func curatorNameFromCronJobName(name string) string {
	return strings.TrimPrefix(name, "curator-")
}

func cronJobNameFromCuratorName(name string) string {
	return "curator-" + name
}

func cronJobKeyToCuratorKey(key string) string {
	keyParts := strings.Split(key, "/")
	return keyParts[0] + "/" + strings.TrimPrefix(keyParts[1], "curator-")
}

func curatorKeyToCronJobKey(key string) string {
	keyParts := strings.Split(key, "/")
	return keyParts[0] + "/curator-" + keyParts[1]
}

func (c *Operator) handleDeleteCronJob(obj interface{}) {
	if ps := c.curatorForCronJob(obj); ps != nil {
		c.enqueue(ps)
	}
}

func (c *Operator) handleAddCronJob(obj interface{}) {
	if ps := c.curatorForCronJob(obj); ps != nil {
		c.enqueue(ps)
	}
}

func (c *Operator) handleUpdateCronJob(oldo, curo interface{}) {
	old := oldo.(*v2alpha1.CronJob)
	cur := curo.(*v2alpha1.CronJob)

	c.logger.Log("msg", "update handler", "old", old.ResourceVersion, "cur", cur.ResourceVersion)

	// Periodic resync may resend the deployment without changes in-between.
	// Also breaks loops created by updating the resource ourselves.
	if old.ResourceVersion == cur.ResourceVersion {
		return
	}

	if ps := c.curatorForCronJob(cur); ps != nil {
		c.enqueue(ps)
	}
}

func (c *Operator) sync(key string) error {
	obj, exists, err := c.curatInf.GetIndexer().GetByKey(key)
	if err != nil {
		return err
	}
	if !exists {
		// TODO(fabxc): we want to do server side deletion due to the variety of
		// resources we create.
		// Doing so just based on the deletion event is not reliable, so
		// we have to garbage collect the controller-created resources in some other way.
		//
		// Let's rely on the index key matching that of the created configmap and CronJob for now.
		// This does not work if we delete Curator resources as the
		// controller is not running – that could be solved via garbage collection later.
		return c.destroyCurator(key)
	}

	p := obj.(*v1alpha1.Curator)
	if p.Spec.Paused {
		return nil
	}

	c.logger.Log("msg", "sync curator", "key", key)

	if err := c.createConfig(p); err != nil {
		return errors.Wrap(err, "creating config failed")
	}

	// Create Secret if it doesn't exist.
	s, err := makeEmptyConfig(p.Name)
	if err != nil {
		return errors.Wrap(err, "generating empty config secret failed")
	}
	if _, err := c.kclient.Core().Secrets(p.Namespace).Create(s); err != nil && !apierrors.IsAlreadyExists(err) {
		return errors.Wrap(err, "creating empty config file failed")
	}

	cjClient := c.kclient.BatchV2alpha1().CronJobs(p.Namespace)
	// Ensure we have a CronJob running Curator deployed.
	obj, exists, err = c.cjInf.GetIndexer().GetByKey(curatorKeyToCronJobKey(key))
	if err != nil {
		return errors.Wrap(err, "retrieving cronjob failed")
	}

	if !exists {
		cj, err := makeCronJob(*p, nil, &c.config)
		if err != nil {
			return errors.Wrap(err, "creating cronjob failed")
		}
		if _, err := cjClient.Create(cj); err != nil {
			return errors.Wrap(err, "creating cronjob failed")
		}
		return nil
	}
	cj, err := makeCronJob(*p, obj.(*v2alpha1.CronJob), &c.config)
	if err != nil {
		return errors.Wrap(err, "updating cronjob failed")
	}
	if _, err := cjClient.Update(cj); err != nil {
		return errors.Wrap(err, "updating cronjob failed")
	}

	return nil
}

// CuratorStatus evaluates the current status of a Curator deployment with respect
// to its specified resource object. It return the status and a list of pods that
// are not updated.
func CuratorStatus(kclient kubernetes.Interface, p *v1alpha1.Curator) (*v1alpha1.CuratorStatus, []v2alpha1.CronJob, error) {
	res := &v1alpha1.CuratorStatus{
		Paused: p.Spec.Paused,
	}

	jobs, err := kclient.BatchV2alpha1().CronJobs(p.Namespace).List(ListOptions(p.Name))
	if err != nil {
		return nil, nil, errors.Wrap(err, "retrieving cronjob of failed")
	}

	var oldJobs []v2alpha1.CronJob
	for _, job := range jobs.Items {
		res.LastScheduleTime = job.Status.LastScheduleTime
		spec, _ := makeCronJobSpec(*p, nil)
		if needsUpdate(&job.Spec.JobTemplate.Spec.Template, spec.JobTemplate.Spec.Template) {
			oldJobs = append(oldJobs, job)
		}
		break
	}

	return res, oldJobs, nil
}

// needsUpdate checks whether the given pod conforms with the pod template spec
// for various attributes that are influenced by the Curator TPR settings.
func needsUpdate(pod *v1.PodTemplateSpec, tmpl v1.PodTemplateSpec) bool {
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

	return false
}

func ListOptions(name string) metav1.ListOptions {
	return metav1.ListOptions{
		LabelSelector: fields.SelectorFromSet(fields.Set(map[string]string{
			"app":     "curator",
			"curator": name,
		})).String(),
	}
}

func (c *Operator) destroyCurator(key string) error {
	cjKey := curatorKeyToCronJobKey(key)
	obj, exists, err := c.cjInf.GetStore().GetByKey(cjKey)
	if err != nil {
		return errors.Wrap(err, "retrieving cronjob from cache failed")
	}
	if !exists {
		return nil
	}
	cj := obj.(*v2alpha1.CronJob)
	*cj.Spec.Suspend = true

	// Update the replica count to 0 and wait for all pods to be deleted.
	cjClient := c.kclient.BatchV2alpha1().CronJobs(cj.Namespace)

	if _, err := cjClient.Update(cj); err != nil {
		return errors.Wrap(err, "updating cronjob for suspend failed")
	}

	podClient := c.kclient.Core().Pods(cj.Namespace)

	// TODO(fabxc): temprorary solution until CronJob status provides necessary info to know
	// whether scale-down completed.
	for {
		pods, err := podClient.List(ListOptions(curatorNameFromCronJobName(cj.Name)))
		if err != nil {
			return errors.Wrap(err, "retrieving pods of cronjob failed")
		}
		if len(pods.Items) == 0 {
			break
		}
		time.Sleep(50 * time.Millisecond)
	}

	// CronJob scaled down, we can delete it.
	if err := cjClient.Delete(cj.Name, nil); err != nil {
		return errors.Wrap(err, "deleting cronjob failed")
	}

	// Delete the auto-generate configuration.
	// TODO(fabxc): add an ownerRef at creation so we don't delete Secrets
	// manually created for Curator servers with no ServiceMonitor selectors.
	s := c.kclient.Core().Secrets(cj.Namespace)
	secret, err := s.Get(cj.Name, metav1.GetOptions{})
	if err != nil && !apierrors.IsNotFound(err) {
		return errors.Wrap(err, "retrieving config Secret failed")
	}
	if apierrors.IsNotFound(err) {
		// Secret does not exist so nothing to clean up
		return nil
	}

	value, found := secret.Labels[managedByOperatorLabel]
	if found && value == managedByOperatorLabelValue {
		if err := s.Delete(cj.Name, nil); err != nil {
			return errors.Wrap(err, "deleting config Secret failed")
		}
	}

	return nil
}

func (c *Operator) createConfig(p *v1alpha1.Curator) error {

	sClient := c.kclient.CoreV1().Secrets(p.Namespace)

	// Update secret based on the most recent configuration.
	conf, err := generateConfig(p)
	if err != nil {
		return errors.Wrap(err, "generating config failed")
	}

	actions, err := generateActions(p)
	if err != nil {
		return errors.Wrap(err, "generating actions failed")
	}

	// TODO(galexrt)
	s, err := makeConfigSecret(p.Name)
	if err != nil {
		return errors.Wrap(err, "generating base secret failed")
	}
	s.ObjectMeta.Annotations = map[string]string{
		"generated": "true",
	}
	s.Data[configFilename] = []byte(conf)
	s.Data[actionsFilename] = []byte(actions)

	curSecret, err := sClient.Get(s.Name, metav1.GetOptions{})
	if apierrors.IsNotFound(err) {
		c.logger.Log("msg", "creating configuration")
		_, err = sClient.Create(s)
		return err
	}

	okay := false

	generatedConf := s.Data[configFilename]
	curConfig, curConfigFound := curSecret.Data[configFilename]
	if curConfigFound {
		if bytes.Equal(curConfig, generatedConf) {
			c.logger.Log("msg", "updating config skipped, no configuration change")
			okay = true
		} else {
			c.logger.Log("msg", "current config has changed")
		}
	} else {
		c.logger.Log("msg", "no current config found", "currentConfigFound", curConfigFound)
	}

	generatedActions := s.Data[actionsFilename]
	curActions, curActionsFound := curSecret.Data[actionsFilename]
	if curActionsFound {
		if bytes.Equal(curActions, generatedActions) {
			c.logger.Log("msg", "updating actions skipped, no configuration change")
			if okay {
				return nil
			}
		} else {
			c.logger.Log("msg", "current actions has changed")
		}
	} else {
		c.logger.Log("msg", "no current actions found", "currentActionsFound", curActionsFound)
	}

	c.logger.Log("msg", "updating configuration")
	_, err = sClient.Update(s)
	return err
}

func (c *Operator) createTPRs() error {
	tprs := []*extensionsobj.ThirdPartyResource{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: tprCurator,
			},
			Versions: []extensionsobj.APIVersion{
				{Name: v1alpha1.TPRVersion},
			},
			Description: "Managed Curator instance(s)",
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
	return k8sutil.WaitForTPRReady(c.kclient.CoreV1().RESTClient(), v1alpha1.TPRGroup, v1alpha1.TPRVersion, v1alpha1.TPRCuratorName)
}
