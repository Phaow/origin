package storage

import (
	"context"
	"reflect"

	g "github.com/onsi/ginkgo/v2"
	o "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	e2e "k8s.io/kubernetes/test/e2e/framework"

	configv1 "github.com/openshift/api/config/v1"
	ocpv1 "github.com/openshift/api/config/v1"
	exutil "github.com/openshift/origin/test/extended/util"
	apiextensionsclientset "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
)

const (
	clusterCSISnapshotOperatorNs = "openshift-cluster-storage-operator"
	snapshotWebhookSecretName    = "csi-snapshot-webhook-secret"
	snapshotWebhookDeployName    = "csi-snapshot-webhook"
)

// This is [Serial] because it deletes the csi-snapshot-webhook-secret
var _ = g.Describe("[sig-storage][Feature:Cluster-CSI-Snapshot-Controller-Operator][Serial][apigroup:operator.openshift.io]", func() {
	defer g.GinkgoRecover()
	var oc = exutil.NewCLIWithoutNamespace("storage-csi-snapshot-operator")

	g.BeforeEach(func() {
		// Skip if CSISnapshot CO is not enabled
		if CSISnapshotEnabled, _ := exutil.IsCapabilityEnabled(oc, configv1.ClusterVersionCapabilityCSISnapshot); !CSISnapshotEnabled {
			g.Skip("Skip for CSISnapshot capability is not enabled on the test cluster!")
		}

		controlPlaneTopology, err := exutil.GetControlPlaneTopology(oc)
		o.Expect(err).NotTo(o.HaveOccurred())

		// HyperShift the CSISnapshot controllers runs on management cluster
		if *controlPlaneTopology == ocpv1.ExternalTopologyMode {
			g.Skip("Clusters with external control plane topology do not run PerformanceProfile Controller")
		}
	})

	g.AfterEach(func() {
		WaitForCSOHealthy(oc)
	})

	g.It("should restart webhook Pods if csi-snapshot-webhook-secret expiry annotation is changed", func() {

		g.By("# Get the csiSnapshotWebhook annotations")
		csiSnapshotWebhookAnnotationsOri := exutil.GetDeploymentTemplateAnnotations(oc, snapshotWebhookDeployName, clusterCSISnapshotOperatorNs)

		g.By("# Modify the csi-snapshot-webhook-secret expiry annotation")
		defer func() {
			if exutil.WaitForDeploymentReady(oc, snapshotWebhookDeployName, clusterCSISnapshotOperatorNs) != nil {
				e2e.Failf("The csiSnapshotWebhook was not recovered ready")
			}
		}()
		o.Expect(oc.AsAdmin().Run("annotate").Args("-n", clusterCSISnapshotOperatorNs, "secret", snapshotWebhookSecretName,
			"service.alpha.openshift.io/expiry-", "service.beta.openshift.io/expiry-").Execute()).NotTo(o.HaveOccurred())

		g.By("# Check the webhook Pods were restarted")
		o.Eventually(func() bool {
			csiSnapshotWebhookAnnotationsCurrent := exutil.GetDeploymentTemplateAnnotations(oc, snapshotWebhookDeployName, clusterCSISnapshotOperatorNs)
			return reflect.DeepEqual(csiSnapshotWebhookAnnotationsOri, csiSnapshotWebhookAnnotationsCurrent)
		}).WithTimeout(defaultMaxWaitingTime).WithPolling(defaultPollingTime).Should(o.BeFalse(), "The csiSnapshotWebhook was not updated")
	})

	g.It("should restart webhook Pods if csi-snapshot-webhook-secret is deleted", func() {

		g.By("# Get the csiSnapshotWebhook annotations")
		csiSnapshotWebhookAnnotationsOri := exutil.GetDeploymentTemplateAnnotations(oc, snapshotWebhookDeployName, clusterCSISnapshotOperatorNs)

		g.By("# Delete the csi-snapshot-webhook-secret")
		defer func() {
			if exutil.WaitForDeploymentReady(oc, snapshotWebhookDeployName, clusterCSISnapshotOperatorNs) != nil {
				e2e.Failf("The csiSnapshotWebhook was not recovered ready")
			}
		}()
		o.Expect(oc.AsAdmin().Run("delete").Args("-n", clusterCSISnapshotOperatorNs, "secret", snapshotWebhookSecretName).Execute()).NotTo(o.HaveOccurred())

		g.By("# Check the webhook Pods were restarted")
		o.Eventually(func() bool {
			csiSnapshotWebhookAnnotationsCurrent := exutil.GetDeploymentTemplateAnnotations(oc, snapshotWebhookDeployName, clusterCSISnapshotOperatorNs)
			return reflect.DeepEqual(csiSnapshotWebhookAnnotationsOri, csiSnapshotWebhookAnnotationsCurrent)
		}).WithTimeout(defaultMaxWaitingTime).WithPolling(defaultPollingTime).Should(o.BeFalse(), "The csiSnapshotWebhook was not updated")
	})
})

var _ = g.Describe("[sig-storage][Feature:VolumeGroupSnapshot][apigroup:operator.openshift.io] Cluster-CSI-Snapshot-Controller-Operator", func() {
	defer g.GinkgoRecover()
	var oc = exutil.NewCLIWithoutNamespace("storage-csi-snapshot-operator")

	g.BeforeEach(func() {
		// Skip if CSISnapshot CO is not enabled
		if CSISnapshotEnabled, _ := exutil.IsCapabilityEnabled(oc, configv1.ClusterVersionCapabilityCSISnapshot); !CSISnapshotEnabled {
			g.Skip("Skip for CSISnapshot capability is not enabled on the test cluster!")
		}

		controlPlaneTopology, err := exutil.GetControlPlaneTopology(oc)
		o.Expect(err).NotTo(o.HaveOccurred())

		// HyperShift the CSISnapshot controllers runs on management cluster
		if *controlPlaneTopology == ocpv1.ExternalTopologyMode {
			g.Skip("Clusters with external control plane topology do not run PerformanceProfile Controller")
		}

		// TODO: Remove this after the VolumeGroupSnapshot is GA
		if !exutil.IsTechPreviewNoUpgrade(oc) {
			g.Skip("this test is only expected to work with TechPreviewNoUpgrade clusters")
		}
	})

	g.It("should enable the VolumeGroupSnapshot for snapshot controller and snapshot webhook", func() {

		g.By("# Check the snapshot controller VolumeGroupSnapshot args enabled")
		csiSnapshotControllerArgs := exutil.GetDeploymentTemplateAnnotations(oc, snapshotWebhookDeployName, clusterCSISnapshotOperatorNs)
		o.Expect(csiSnapshotControllerArgs).Should(o.HaveExactElements("--enable-volume-group-snapshots"), "The snapshot controller VolumeGroupSnapshot args is not enabled")

		g.By("# Check the snapshot webhook VolumeGroupSnapshot args enabled")
		csiSnapshotWebhookArgs := exutil.GetDeploymentSpecifiedContainerArgs(oc, snapshotWebhookDeployName, clusterCSISnapshotOperatorNs, "webhook")
		o.Expect(csiSnapshotWebhookArgs).Should(o.HaveExactElements("--enable-volume-group-snapshot-webhook"), "The snapshot webhook VolumeGroupSnapshot args is not enabled")
	})

	g.It("should create the VolumeGroupSnapshot CRDs", func() {

		g.By("# Check the VolumeGroupSnapshot CRDs created")
		crdClient := apiextensionsclientset.NewForConfigOrDie(oc.AdminConfig())
		crdList, getCRDsErr := crdClient.ApiextensionsV1().CustomResourceDefinitions().List(context.Background(), metav1.ListOptions{})
		o.Expect(getCRDsErr).ShouldNot(o.HaveOccurred(), "Failed to get CRDs list")
		e2e.Logf("%s", crdList.Kind)
	})
})
