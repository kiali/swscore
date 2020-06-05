package kubetest

import "github.com/kiali/kiali/kubernetes"

func (o *K8SClientMock) CreateIstioObject(api, namespace, resourceType, json string) (kubernetes.IstioObject, error) {
	args := o.Called(api, namespace, resourceType, json)
	return args.Get(0).(kubernetes.IstioObject), args.Error(1)
}

func (o *K8SClientMock) DeleteIstioObject(api, namespace, objectType, objectName string) error {
	args := o.Called(api, namespace, objectType, objectName)
	return args.Error(0)
}

func (o *K8SClientMock) GetAuthorizationDetails(namespace string) (*kubernetes.RBACDetails, error) {
	args := o.Called(namespace)
	return args.Get(0).(*kubernetes.RBACDetails), args.Error(1)
}

func (o *K8SClientMock) GetClusterRbacConfigs() ([]kubernetes.IstioObject, error) {
	args := o.Called()
	return args.Get(0).([]kubernetes.IstioObject), args.Error(1)
}

func (o *K8SClientMock) GetClusterRbacConfig(policyName string) (kubernetes.IstioObject, error) {
	args := o.Called(policyName)
	return args.Get(0).(kubernetes.IstioObject), args.Error(1)
}

func (o *K8SClientMock) GetDestinationRules(namespace string, serviceName string) ([]kubernetes.IstioObject, error) {
	args := o.Called(namespace, serviceName)
	return args.Get(0).([]kubernetes.IstioObject), args.Error(1)
}

func (o *K8SClientMock) GetIstioConfigMap() (*kubernetes.IstioMeshConfig, error) {
	args := o.Called()
	return args.Get(0).(*kubernetes.IstioMeshConfig), args.Error(1)
}

func (o *K8SClientMock) GetIstioObject(namespace string, resourceType string, object string) (kubernetes.IstioObject, error) {
	args := o.Called(namespace, resourceType, object)
	return args.Get(0).(kubernetes.IstioObject), args.Error(1)
}

func (o *K8SClientMock) GetIstioObjects(namespace, resourceType, labelSelector string) ([]kubernetes.IstioObject, error) {
	args := o.Called(namespace, resourceType, labelSelector)
	return args.Get(0).([]kubernetes.IstioObject), args.Error(1)
}

func (o *K8SClientMock) GetMeshPolicies() ([]kubernetes.IstioObject, error) {
	args := o.Called()
	return args.Get(0).([]kubernetes.IstioObject), args.Error(1)
}

func (o *K8SClientMock) GetMeshPolicy(policyName string) (kubernetes.IstioObject, error) {
	args := o.Called(policyName)
	return args.Get(0).(kubernetes.IstioObject), args.Error(1)
}

func (o *K8SClientMock) GetVirtualServices(namespace string, serviceName string) ([]kubernetes.IstioObject, error) {
	args := o.Called(namespace, serviceName)
	return args.Get(0).([]kubernetes.IstioObject), args.Error(1)
}

func (o *K8SClientMock) UpdateIstioObject(api, namespace, resourceType, name, jsonPatch string) (kubernetes.IstioObject, error) {
	args := o.Called(api, namespace, resourceType, name, jsonPatch)
	return args.Get(0).(kubernetes.IstioObject), args.Error(1)
}
