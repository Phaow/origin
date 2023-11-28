package storage

import (
	"strings"

	g "github.com/onsi/ginkgo/v2"
	o "github.com/onsi/gomega"

	configv1 "github.com/openshift/api/config/v1"
	exutil "github.com/openshift/origin/test/extended/util"
)

// This is [Serial] because it deletes the csi-snapshot-webhook-secret
var _ = g.Describe("[sig-storage][Feature:Cluster-CSI-Snapshot-Controller-Operator][Serial][apigroup:operator.openshift.io]", func() {
	defer g.GinkgoRecover()
	var (
		oc = exutil.NewCLIWithoutNamespace("storage-csi-snapshot-operator")
	)

	g.BeforeEach(func() {
		// Skip if CSISnapshot CO is not enabled
		if CSISnapshotEnabled, _ := exutil.IsCapabilityEnabled(oc, configv1.ClusterVersionCapabilityCSISnapshot); !CSISnapshotEnabled {
			g.Skip("Skip for CSISnapshot capability is not enabled on the test cluster!")
		}
	})

	g.AfterEach(func() {
		WaitForCSOHealthy(oc)
	})

	g.It("should restart webhook Pods if csi-snapshot-webhook-secret expiry annotation changed", func() {

		var (
			clusterCSISnapshotOperatorNs = "openshift-cluster-storage-operator"
			snapshotWebhookSecretName    = "csi-snapshot-webhook-secret"
			csiSnapshotWebhook           = NewDeployment(SetDeploymentName("csi-snapshot-webhook"), SetDeploymentNamespace(clusterCSISnapshotOperatorNs), SetDeploymentApplabel("app=csi-snapshot-webhook"))
		)

		g.By("# Get the csiSnapshotWebhook resourceVersion")
		csiSnapshotWebhookResourceVersionOri := csiSnapshotWebhook.GetSpecifiedJSONPathValue(oc, "{.metadata.resourceVersion}")

		g.By("# Modify the csi-snapshot-webhook-secret expiry annotation")
		defer csiSnapshotWebhook.WaitForReady(oc.AsAdmin())
		o.Expect(oc.AsAdmin().Run("annotate").Args("-n", clusterCSISnapshotOperatorNs, "secret", snapshotWebhookSecretName,
			"service.alpha.openshift.io/expiry-", "service.beta.openshift.io/expiry-").Execute()).NotTo(o.HaveOccurred())

		g.By("# Check the webhook Pods should be restarted")
		o.Eventually(func() bool {
			csiSnapshotWebhookResourceVersionCurrent := csiSnapshotWebhook.GetSpecifiedJSONPathValue(oc, "{.metadata.resourceVersion}")
			return strings.EqualFold(csiSnapshotWebhookResourceVersionOri, csiSnapshotWebhookResourceVersionCurrent)
		}, defaultMaxWaitingTime, defaultMaxWaitingTime/defaultIterationTimes).Should(o.BeFalse(), "The csiSnapshotWebhook deployment resourceVersion doesn't update")
		csiSnapshotWebhook.WaitForReady(oc.AsAdmin())
	})

	g.It("should restart webhook Pods if csi-snapshot-webhook-secret deleted", func() {

		var (
			clusterCSISnapshotOperatorNs = "openshift-cluster-storage-operator"
			snapshotWebhookSecretName    = "csi-snapshot-webhook-secret"
			csiSnapshotWebhook           = NewDeployment(SetDeploymentName("csi-snapshot-webhook"), SetDeploymentNamespace(clusterCSISnapshotOperatorNs), SetDeploymentApplabel("app=csi-snapshot-webhook"))
		)

		g.By("# Get the csiSnapshotWebhook resourceVersion")
		csiSnapshotWebhookResourceVersionOri := csiSnapshotWebhook.GetSpecifiedJSONPathValue(oc, "{.metadata.resourceVersion}")

		g.By("# Delete the csi-snapshot-webhook-secret")
		defer csiSnapshotWebhook.WaitForReady(oc.AsAdmin())
		o.Expect(oc.AsAdmin().Run("delete").Args("-n", clusterCSISnapshotOperatorNs, "secret", snapshotWebhookSecretName).Execute()).NotTo(o.HaveOccurred())

		g.By("# Check the webhook Pods should be restarted")
		o.Eventually(func() bool {
			csiSnapshotWebhookResourceVersionCurrent := csiSnapshotWebhook.GetSpecifiedJSONPathValue(oc, "{.metadata.resourceVersion}")
			return strings.EqualFold(csiSnapshotWebhookResourceVersionOri, csiSnapshotWebhookResourceVersionCurrent)
		}, defaultMaxWaitingTime, defaultMaxWaitingTime/defaultIterationTimes).Should(o.BeFalse(), "The csiSnapshotWebhook deployment resourceVersion doesn't update")
		csiSnapshotWebhook.WaitForReady(oc.AsAdmin())
	})
})
