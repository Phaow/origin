package storage

import (
	"context"
	"math/rand"
	"time"

	o "github.com/onsi/gomega"
	configv1 "github.com/openshift/api/config/v1"
	clusteroperatorhelpers "github.com/openshift/library-go/pkg/config/clusteroperator/v1helpers"
	exutil "github.com/openshift/origin/test/extended/util"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	e2e "k8s.io/kubernetes/test/e2e/framework"
)

// GetRandomString gets a random string of 8 byte
func GetRandomString() string {
	chars := "abcdefghijklmnopqrstuvwxyz0123456789"
	seed := rand.New(rand.NewSource(time.Now().UnixNano()))
	buffer := make([]byte, 8)
	for index := range buffer {
		buffer[index] = chars[seed.Intn(len(chars))]
	}
	return string(buffer)
}

// IsCSOHealthy checks whether the Cluster Storage Operator is healthy
func IsCSOHealthy(oc *exutil.CLI) (bool, error) {
	// CSO healthyStatus:[degradedStatus:False, progressingStatus:False, availableStatus:True, upgradeableStatus:True]
	clusterStorageOperator, getOperatorErr := oc.AdminConfigClient().ConfigV1().ClusterOperators().Get(context.Background(), "storage", metav1.GetOptions{})
	if getOperatorErr != nil {
		e2e.Logf("Error getting storage operator: %v", getOperatorErr)
		return false, getOperatorErr
	}
	return clusteroperatorhelpers.IsStatusConditionTrue(clusterStorageOperator.Status.Conditions, configv1.OperatorAvailable) &&
		clusteroperatorhelpers.IsStatusConditionTrue(clusterStorageOperator.Status.Conditions, configv1.OperatorUpgradeable) &&
		clusteroperatorhelpers.IsStatusConditionFalse(clusterStorageOperator.Status.Conditions, configv1.OperatorDegraded) &&
		clusteroperatorhelpers.IsStatusConditionFalse(clusterStorageOperator.Status.Conditions, configv1.OperatorProgressing), nil
}

// WaitForCSOHealthy waits for Cluster Storage Operator become healthy
func WaitForCSOHealthy(oc *exutil.CLI) {
	o.Eventually(func() bool {
		IsCSOHealthy, getCSOHealthyErr := IsCSOHealthy(oc)
		if getCSOHealthyErr != nil {
			e2e.Logf(`Get CSO status failed of: "%v", try again`, getCSOHealthyErr)
		}
		return IsCSOHealthy
	}, defaultMaxWaitingTime, defaultMaxWaitingTime/defaultIterationTimes).Should(o.BeTrue(), "Waiting for CSO become healthy timeout")
}
