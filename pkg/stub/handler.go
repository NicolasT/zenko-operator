package stub

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/NicolasT/zenko-operator/pkg/apis/zenko/v1alpha1"

	"github.com/operator-framework/operator-sdk/pkg/sdk"
	"github.com/sirupsen/logrus"
	//corev1 "k8s.io/api/core/v1"
	//"k8s.io/apimachinery/pkg/api/errors"
	//metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	//"k8s.io/apimachinery/pkg/runtime/schema"

	"k8s.io/helm/pkg/chartutil"
	"k8s.io/helm/pkg/helm"
	"k8s.io/helm/pkg/storage/driver"
	"k8s.io/helm/pkg/strvals"

	"github.com/ghodss/yaml"

	"k8s.io/apimachinery/pkg/util/uuid"

	"crypto/sha256"
)

func NewHandler(client helm.Interface) sdk.Handler {
	return &Handler{
		client,
	}
}

type Handler struct {
	client helm.Interface
}

func (h *Handler) Handle(ctx context.Context, event sdk.Event) error {
	switch o := event.Object.(type) {
	case *v1alpha1.Zenko:
		if event.Deleted {
			// We're currently not setting OwnerReferences etc
			return h.deleteZenko(o)
		}
		return h.syncZenko(o)
	}
	return nil
}

func (h *Handler) syncZenko(o *v1alpha1.Zenko) error {
	logrus.Debugf("Handling Zenko: %s/%s", o.ObjectMeta.Namespace, o.ObjectMeta.Name)

	chartPath := "charts/zenko-" + o.Spec.AppVersion
	if _, err := os.Stat(chartPath); err != nil {
		return fmt.Errorf("Unsupported appVersion")
	}

	releaseName := o.ObjectMeta.Name

	releaseHistory, err := h.client.ReleaseHistory(releaseName, helm.WithMaxHistory(1))
	if err == nil {
		previousReleaseNamespace := releaseHistory.Releases[0].Namespace
		if previousReleaseNamespace != o.ObjectMeta.Namespace {
			// TODO Set status or whatever?
			logrus.Warningf("Duplicate deployment name %v in namespace %v, release %v already exists in namespace %v", o.ObjectMeta.Name, o.ObjectMeta.Namespace, releaseName, previousReleaseNamespace)
			return nil
		}
	}

	if err != nil && strings.Contains(err.Error(), driver.ErrReleaseNotFound(releaseName).Error()) {
		return h.installZenko(o)
	}

	return h.updateZenko(o)
}

func (h *Handler) installZenko(o *v1alpha1.Zenko) error {
	logrus.Infof("Installing Zenko: %s/%s", o.ObjectMeta.Namespace, o.ObjectMeta.Name)

	releaseName := o.ObjectMeta.Name // TODO Duplication

	vals, err := calculateValues(o)
	if err != nil {
		return err
	}

	// https://github.com/helm/helm/blob/71629f046adceadcea20d9515797f5887eca7dc8/cmd/helm/install.go#L225
	chartRequested, err := chartutil.Load("charts/zenko-" + o.Spec.AppVersion)
	if err != nil {
		return err
	}

	if _, err := chartutil.LoadRequirements(chartRequested); err != nil {
		return err
	}

	o.Status.InstanceID = uuid.NewUUID()
	err = sdk.Update(o)
	if err != nil {
		return fmt.Errorf("failed to update zenko status: %v", err)
	}

	//TODO Run all of this in a Goroutine and set an Event on the object
	//when deployment finished
	_, err = h.client.InstallReleaseFromChart(
		chartRequested,
		o.ObjectMeta.Namespace,
		helm.ValueOverrides(vals),
		helm.ReleaseName(releaseName),
		helm.InstallDryRun(false),
		helm.InstallReuseName(false),
		helm.InstallDisableHooks(false),
		helm.InstallTimeout(300),
		helm.InstallWait(false))

	if err != nil {
		return err
	}

	deployedConfigurationHash := fmt.Sprintf("%x", sha256.Sum256(vals))

	o.Status.DeployedVersion = o.Spec.AppVersion
	o.Status.DeployedConfigurationHash = deployedConfigurationHash
	err = sdk.Update(o)
	if err != nil {
		return err
	}

	//TODO Emit an event on the Zenko object

	return err
}

func (h *Handler) updateZenko(o *v1alpha1.Zenko) error {
	mustUpdate := false

	if o.Status.DeployedVersion != o.Spec.AppVersion {
		mustUpdate = true
	}
	if !mustUpdate {
		newValues, err := calculateValues(o)
		if err != nil {
			return err
		}

		configHash := fmt.Sprintf("%x", sha256.Sum256(newValues))
		if configHash != o.Status.DeployedConfigurationHash {
			mustUpdate = true
		}
	}

	if !mustUpdate {
		logrus.Debugf("Nothing to be done")
		return nil
	}

	logrus.Infof("Updating Zenko")

	releaseName := o.ObjectMeta.Name

	newValues, err := calculateValues(o)
	if err != nil {
		return err
	}

	// https://github.com/helm/helm/blob/71629f046adceadcea20d9515797f5887eca7dc8/cmd/helm/upgrade.go#L154
	if _, err := chartutil.Load("charts/zenko-" + o.Spec.AppVersion); err != nil {
		//TODO Much more code here in the original!
		return err
	}

	_, err = h.client.UpdateRelease(
		releaseName,
		"charts/zenko-"+o.Spec.AppVersion,
		helm.UpdateValueOverrides(newValues),
		helm.UpgradeDryRun(false),
		helm.UpgradeRecreate(false),
		helm.UpgradeForce(false),
		helm.UpgradeDisableHooks(false),
		helm.UpgradeTimeout(300),
		helm.ResetValues(true),
		helm.ReuseValues(false),
		helm.UpgradeWait(false))
	if err != nil {
		return err
	}

	deployedConfigurationHash := fmt.Sprintf("%x", sha256.Sum256(newValues))

        // TODO Only set this when it's actually deployed, or something
	o.Status.DeployedVersion = o.Spec.AppVersion
	o.Status.DeployedConfigurationHash = deployedConfigurationHash
	err = sdk.Update(o)

	//TODO Emit an event on the Zenko object

	return err
}

func (h *Handler) deleteZenko(o *v1alpha1.Zenko) error {
	logrus.Infof("Deleting Zenko: %s/%s", o.ObjectMeta.Namespace, o.ObjectMeta.Name)

	opts := []helm.DeleteOption{
		helm.DeleteDryRun(false),
		helm.DeleteDisableHooks(false),
		helm.DeletePurge(true),
		helm.DeleteTimeout(300),
	}
	_, err := h.client.DeleteRelease(o.ObjectMeta.Name, opts...)
	return err
}

func calculateValues(o *v1alpha1.Zenko) ([]byte, error) {
	base := map[string]interface{}{}

	nodeCount := o.Spec.NodeCount

	nodeCountSettings := []string{
		"nodeCount",
		"cloudserver.replicaCount",
		"cloudserver.mongodb.replicas",
		"backbeat.replication.dataProcessor.replicaCount",
		"backbeat.replication.statusProcessor.replicaCount",
		"backbeat.lifecycle.bucketProcessor.replicaCount",
		"backbeat.lifecycle.objectProcessor.replicaCount",
		"backbeat.garbageCollector.consumer.replicaCount",
		"backbeat.mongodb.replicas",
		"zenko-nfs.mongodb.replicas",
		"mongodb-replicaset.replicas",
		"zenko-queue.replicas",
		//"zenko-queue.configurationOverrides['num.partitions']",
		"zenko-quorum.replicaCount",
		"redis-ha.replicas.servers",
		"redis-ha.replicas.sentinels",
	}

	for _, setting := range nodeCountSettings {
		arg := fmt.Sprintf("%s=%d", setting, nodeCount)
		if err := strvals.ParseInto(arg, base); err != nil {
			return []byte{}, fmt.Errorf("failed parsing %v: %s", arg, err)
		}
	}

	return yaml.Marshal(base)
}
