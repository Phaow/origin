package storage

import (
	"fmt"
	"strings"
	"time"

	o "github.com/onsi/gomega"
	exutil "github.com/openshift/origin/test/extended/util"
	e2e "k8s.io/kubernetes/test/e2e/framework"
)

// Define test waiting time const
const (
	defaultMaxWaitingTime    = 300 * time.Second
	defaultIterationTimes    = 20
	longerMaxWaitingTime     = 15 * time.Minute
	moreLongerMaxWaitingTime = 30 * time.Minute
	longestMaxWaitingTime    = 1 * time.Hour
)

// Deployment workload definition
type Deployment struct {
	Name             string
	Namespace        string
	Replicas         string
	AppLabel         string
	MountPath        string
	PvcName          string
	Template         string
	VolumeType       string
	TypePath         string
	MaxWaitReadyTime time.Duration
}

// DeployOption uses function option mode to change the default value of Deployment attributes, eg. name, replicas
type DeployOption func(*Deployment)

// SetDeploymentName replaces the default value of Deployment Name
func SetDeploymentName(name string) DeployOption {
	return func(deploy *Deployment) {
		deploy.Name = name
	}
}

// SetDeploymentTemplate replaces the default value of Deployment Template
func SetDeploymentTemplate(template string) DeployOption {
	return func(deploy *Deployment) {
		deploy.Template = template
	}
}

// SetDeploymentNamespace replaces the default value of Deployment Namespace
func SetDeploymentNamespace(namespace string) DeployOption {
	return func(deploy *Deployment) {
		deploy.Namespace = namespace
	}
}

// SetDeploymentReplicas replaces the default value of Deployment Replicas
func SetDeploymentReplicas(replicas string) DeployOption {
	return func(deploy *Deployment) {
		deploy.Replicas = replicas
	}
}

// SetDeploymentApplabel replaces the default value of Deployment AppLabel
func SetDeploymentApplabel(appLabel string) DeployOption {
	return func(deploy *Deployment) {
		deploy.AppLabel = appLabel
	}
}

// SetDeploymentMountpath replaces the default value of Deployment MountPath
func SetDeploymentMountpath(mountPath string) DeployOption {
	return func(deploy *Deployment) {
		deploy.MountPath = mountPath
	}
}

// SetDeploymentPVCName replaces the default value of Deployment PvcName
func SetDeploymentPVCName(pvcname string) DeployOption {
	return func(deploy *Deployment) {
		deploy.PvcName = pvcname
	}
}

// SetDeploymentVolumeType replaces the default value of Deployment volume type
func SetDeploymentVolumeType(volumeType string) DeployOption {
	return func(deploy *Deployment) {
		deploy.VolumeType = volumeType
	}
}

// SetDeploymentVolumeTypePath replaces the default value of Deployment TypePath
func SetDeploymentVolumeTypePath(typePath string) DeployOption {
	return func(deploy *Deployment) {
		deploy.TypePath = typePath
	}
}

// SetDeploymentReplicasNo replaces the default value of Deployment Replicas
func SetDeploymentReplicasNo(replicas string) DeployOption {
	return func(deploy *Deployment) {
		deploy.Replicas = replicas
	}
}

// SetDeploymentMaxWaitReadyTime replaces the default value of Deployment MaxWaitReadyTime
func SetDeploymentMaxWaitReadyTime(maxWaitReadyTime time.Duration) DeployOption {
	return func(deploy *Deployment) {
		deploy.MaxWaitReadyTime = maxWaitReadyTime
	}
}

// NewDeployment creates a new customized Deployment object
func NewDeployment(opts ...DeployOption) Deployment {
	defaultMaxWaitReadyTime := defaultMaxWaitingTime
	deployNameSuffix := GetRandomString()
	defaultDeployment := Deployment{
		Name:             "e2e-" + deployNameSuffix,
		Template:         "deploy-template.yaml",
		Namespace:        "",
		Replicas:         "1",
		AppLabel:         "app=" + "e2e-" + deployNameSuffix,
		MountPath:        "/mnt/storage",
		PvcName:          "",
		VolumeType:       "volumeMounts",
		TypePath:         "mountPath",
		MaxWaitReadyTime: defaultMaxWaitReadyTime,
	}

	for _, o := range opts {
		o(&defaultDeployment)
	}

	return defaultDeployment
}

// GetPodList gets Deployment 'Running' pods list
func (dep *Deployment) GetPodList(oc *exutil.CLI) (podsList []string) {
	selectorLabel := dep.AppLabel
	if !strings.Contains(dep.AppLabel, "=") {
		selectorLabel = "app=" + dep.AppLabel
	}
	o.Eventually(func() bool {
		podsListStr, _ := oc.WithoutNamespace().Run("get").Args("pod", "-n", dep.Namespace, "-l", selectorLabel, "-o=jsonpath={.items[?(@.status.phase==\"Running\")].metadata.name}").Output()
		podsList = strings.Fields(podsListStr)
		return strings.EqualFold(fmt.Sprint(len(podsList)), dep.Replicas)
	}, defaultMaxWaitingTime, defaultMaxWaitingTime/defaultIterationTimes).Should(o.BeTrue(), fmt.Sprintf("Failed to get Deployment %s's ready podlist", dep.Name))
	e2e.Logf("Deployment/%s's ready podlist is: %v", dep.Name, podsList)
	return
}

// GetReplicas gets Replicas of the Deployment
func (dep *Deployment) GetReplicas(oc *exutil.CLI) string {
	replicas, err := oc.AsAdmin().WithoutNamespace().Run("get").Args("deployment", dep.Name, "-n", dep.Namespace, "-o", "jsonpath={.spec.replicas}").Output()
	o.Expect(err).NotTo(o.HaveOccurred())
	return replicas
}

// ScaleReplicas scales Replicas for the Deployment
func (dep *Deployment) ScaleReplicas(oc *exutil.CLI, replicas string) {
	err := oc.WithoutNamespace().Run("scale").Args("Deployment", dep.Name, "--replicas="+replicas, "-n", dep.Namespace).Execute()
	o.Expect(err).NotTo(o.HaveOccurred())
	dep.Replicas = replicas
}

// GetSpecifiedJSONPathValue gets the specified jsonPath value of the Deployment
func (dep *Deployment) GetSpecifiedJSONPathValue(oc *exutil.CLI, jsonPath string) string {
	value, getValueErr := oc.AsAdmin().WithoutNamespace().Run("get").Args("deployment", dep.Name, "-n", dep.Namespace, "-o", fmt.Sprintf("jsonpath=%s", jsonPath)).Output()
	o.Expect(getValueErr).NotTo(o.HaveOccurred())
	e2e.Logf(`Deployment/%s jsonPath->"%s" value is %s`, dep.Name, jsonPath, value)
	return value
}

// Restart restarts the Deployment by rollout restart
func (dep *Deployment) Restart(oc *exutil.CLI) {
	resourceVersionOri := dep.GetSpecifiedJSONPathValue(oc, "{.metadata.resourceVersion}")
	err := oc.WithoutNamespace().Run("rollout").Args("-n", dep.Namespace, "restart", "Deployment", dep.Name).Execute()
	o.Expect(err).NotTo(o.HaveOccurred())
	o.Eventually(func() string {
		return dep.GetSpecifiedJSONPathValue(oc, "{.metadata.resourceVersion}")
	}, defaultMaxWaitingTime, defaultMaxWaitingTime/defaultIterationTimes).ShouldNot(o.Equal(resourceVersionOri), "The deployment resourceVersion doesn't update")
	dep.WaitForReady(oc)
	e2e.Logf("Deployment/%s in namespace %s restart successfully", dep.Name, dep.Namespace)
}

// HardRestart restarts the Deployment by deleting it's pods
func (dep *Deployment) HardRestart(oc *exutil.CLI) {
	o.Expect(oc.WithoutNamespace().Run("delete").Args("-n", dep.Namespace, "pod", "-l", dep.AppLabel).Execute()).NotTo(o.HaveOccurred())
	dep.WaitForReady(oc)
	e2e.Logf("Deployment/%s in namespace %s hard restart successfully", dep.Name, dep.Namespace)
}

// IsReady checks whether the Deployment is ready
func (dep *Deployment) IsReady(oc *exutil.CLI) (bool, error) {
	dep.Replicas = dep.GetReplicas(oc)
	readyReplicas, err := oc.WithoutNamespace().Run("get").Args("deployment", dep.Name, "-n", dep.Namespace, "-o", "jsonpath={.status.availableReplicas}").Output()
	if err != nil {
		return false, err
	}
	if dep.Replicas == "0" && readyReplicas == "" {
		readyReplicas = "0"
	}
	return strings.EqualFold(dep.Replicas, readyReplicas), nil
}

// Describe gets the description of the Deployment
func (dep *Deployment) Describe(oc *exutil.CLI) (string, error) {
	return oc.WithoutNamespace().Run("describe").Args("Deployment", dep.Name, "-n", dep.Namespace).Output()
}

// LongerTime changes dep.MaxWaitReadyTime to LongerMaxWaitingTime
// Used for some Longduration test
func (dep *Deployment) LongerTime() *Deployment {
	newDep := *dep
	newDep.MaxWaitReadyTime = longerMaxWaitingTime
	return &newDep
}

// SpecifiedLongerTime changes dep.MaxWaitReadyTime to specifiedDuring max wait time
// Used for some Longduration test
func (dep *Deployment) SpecifiedLongerTime(specifiedDuring time.Duration) *Deployment {
	newDep := *dep
	newDep.MaxWaitReadyTime = specifiedDuring
	return &newDep
}

// WaitForReady waits for the Deployment become ready
func (dep *Deployment) WaitForReady(oc *exutil.CLI) {
	o.Eventually(func() bool {
		IsDeployReady, getDeployStatusErr := dep.IsReady(oc)
		if getDeployStatusErr != nil {
			e2e.Logf(`Get deployment status failed of: "%v", try again`, getDeployStatusErr)
		}
		return IsDeployReady
	}, dep.MaxWaitReadyTime, dep.MaxWaitReadyTime/defaultIterationTimes).Should(o.BeTrue(), "Waiting for deployment/%s become healthy timeout", dep.Name)
}
