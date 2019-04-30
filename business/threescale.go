package business

import (
	"encoding/json"
	"fmt"
	"sync"

	"k8s.io/apimachinery/pkg/api/errors"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/kiali/kiali/config"
	"github.com/kiali/kiali/kubernetes"
	"github.com/kiali/kiali/log"
	"github.com/kiali/kiali/models"
	"github.com/kiali/kiali/prometheus/internalmetrics"
)

type ThreeScaleService struct {
	k8s kubernetes.IstioClientInterface
}

func (in *ThreeScaleService) GetThreeScaleInfo() (models.ThreeScaleInfo, error) {
	var err error
	promtimer := internalmetrics.GetGoFunctionMetric("business", "ThreeScaleService", "GetThreeScaleInfo")
	defer promtimer.ObserveNow(&err)

	conf := config.Get()
	_, err2 := in.k8s.GetAdapter(conf.IstioNamespace, "adapters", conf.ExternalServices.ThreeScale.AdapterName)
	if err2 != nil {
		if errors.IsNotFound(err2) {
			return models.ThreeScaleInfo{}, nil
		} else {
			return models.ThreeScaleInfo{}, err2
		}
	}
	canCreate, canUpdate, canDelete := getPermissions(in.k8s, conf.IstioNamespace, "adapters", "adapters")
	return models.ThreeScaleInfo{
		Enabled: true,
		Permissions: models.ResourcePermissions{
			Create: canCreate,
			Update: canUpdate,
			Delete: canDelete,
		}}, nil
}

func (in *ThreeScaleService) GetThreeScaleHandlers() (models.ThreeScaleHandlers, error) {
	var err error
	promtimer := internalmetrics.GetGoFunctionMetric("business", "ThreeScaleService", "GetThreeScaleHandlers")
	defer promtimer.ObserveNow(&err)

	return in.getThreeScaleHandlers()
}

func (in *ThreeScaleService) CreateThreeScaleHandler(body []byte) (models.ThreeScaleHandlers, error) {
	var err error
	promtimer := internalmetrics.GetGoFunctionMetric("business", "ThreeScaleService", "CreateThreeScaleHandler")
	defer promtimer.ObserveNow(&err)

	conf := config.Get()

	threeScaleHandler := &models.ThreeScaleHandler{}
	err2 := json.Unmarshal(body, threeScaleHandler)
	if err2 != nil {
		log.Errorf("JSON: %s shows error: %s", string(body), err2)
		err = fmt.Errorf(models.BadThreeScaleHandlerJson)
		return nil, err
	}

	jsonHandler, jsonInstance, err2 := generateJsonHandlerInstance(*threeScaleHandler)
	if err2 != nil {
		log.Error(err2)
		err = fmt.Errorf(models.BadThreeScaleHandlerJson)
		return nil, err
	}

	var wg sync.WaitGroup
	wg.Add(1)

	var errHandler, errInstance error

	go func() {
		defer wg.Done()
		_, errInstance = in.k8s.CreateIstioObject(resourceTypesToAPI["templates"], conf.IstioNamespace, "instances", jsonInstance)
	}()

	// Create handler on main goroutine
	_, errHandler = in.k8s.CreateIstioObject(resourceTypesToAPI["adapters"], conf.IstioNamespace, "handlers", jsonHandler)

	wg.Wait()

	if errHandler != nil {
		return nil, errHandler
	}
	if errInstance != nil {
		return nil, errHandler
	}

	return in.getThreeScaleHandlers()
}

// Private get 3scale handlers to be reused for several public methods
func (in *ThreeScaleService) getThreeScaleHandlers() (models.ThreeScaleHandlers, error) {
	conf := config.Get()
	// Istio config generated from Kiali will be labeled as kiali_wizard
	tsh, err2 := in.k8s.GetAdapters(conf.IstioNamespace, "kiali_wizard")
	if err2 != nil {
		return models.ThreeScaleHandlers{}, err2
	}
	return models.CastThreeScaleHandlers(tsh), nil
}

// It will generate the JSON representing the Handler and Instance that will be used for the ThreeScale Handler
func generateJsonHandlerInstance(handler models.ThreeScaleHandler) (string, string, error) {
	conf := config.Get()
	newHandler := kubernetes.GenericIstioObject{
		TypeMeta: meta_v1.TypeMeta{
			APIVersion: "config.istio.io/v1alpha2",
			Kind:       "handler",
		},
		ObjectMeta: meta_v1.ObjectMeta{
			Name:      handler.Name,
			Namespace: conf.IstioNamespace,
			Labels: map[string]string{
				"kiali_wizard": "threescale-handler",
			},
		},
		Spec: map[string]interface{}{
			"adapter": conf.ExternalServices.ThreeScale.AdapterName,
			"params": map[string]interface{}{
				"service_id":   handler.ServiceId,
				"system_url":   handler.SystemUrl,
				"access_token": handler.AccessToken,
			},
			"connection": map[string]interface{}{
				"address": conf.ExternalServices.ThreeScale.AdapterService + ":" + conf.ExternalServices.ThreeScale.AdapterPort,
			},
		},
	}

	newInstance := kubernetes.GenericIstioObject{
		TypeMeta: meta_v1.TypeMeta{
			APIVersion: "config.istio.io/v1alpha2",
			Kind:       "instance",
		},
		ObjectMeta: meta_v1.ObjectMeta{
			Name:      "threescale-authorization-" + handler.Name,
			Namespace: conf.IstioNamespace,
			Labels: map[string]string{
				"kiali_wizard": "threescale-handler",
			},
		},
		Spec: map[string]interface{}{
			"template": "threescale-authorization",
			"params": map[string]interface{}{
				"subject": map[string]interface{}{
					"user": "request.query_params[\"user_key\"] | request.headers[\"User-Key\"] | \"\"",
					"properties": map[string]interface{}{
						"app_id":  "request.query_params[\"app_id\"] | request.headers[\"App-Id\"] | \"\"",
						"app_key": "request.query_params[\"app_key\"] | request.headers[\"App-Key\"] | \"\"",
					},
				},
				"action": map[string]interface{}{
					"path":   "request.url_path",
					"method": "request.method | \"get\"",
				},
			},
		},
	}

	bHandler, err := json.Marshal(newHandler)
	if err != nil {
		return "", "", err
	}
	bInstance, err := json.Marshal(newInstance)
	if err != nil {
		return "", "", err
	}
	return string(bHandler), string(bInstance), nil
}

func (in *ThreeScaleService) UpdateThreeScaleHandler(handlerName string, body []byte) (models.ThreeScaleHandlers, error) {
	var err error
	promtimer := internalmetrics.GetGoFunctionMetric("business", "ThreeScaleService", "UpdateThreeScaleHandler")
	defer promtimer.ObserveNow(&err)

	threeScaleHandler := &models.ThreeScaleHandler{}
	err2 := json.Unmarshal(body, threeScaleHandler)
	if err2 != nil {
		log.Errorf("JSON: %s shows error: %s", string(body), err2)
		err = fmt.Errorf(models.BadThreeScaleHandlerJson)
		return nil, err
	}

	// Be sure that name inside body is same as used as parameter
	(*threeScaleHandler).Name = handlerName

	// We need the handler structure generated from the ThreeScaleHandler to update it
	jsonUpdatedHandler, _, err2 := generateJsonHandlerInstance(*threeScaleHandler)

	conf := config.Get()

	_, err = in.k8s.UpdateIstioObject(resourceTypesToAPI["adapters"], conf.IstioNamespace, "handlers", handlerName, jsonUpdatedHandler)
	if err != nil {
		return nil, err
	}

	return in.getThreeScaleHandlers()
}

func (in *ThreeScaleService) DeleteThreeScaleHandler(handlerName string) (models.ThreeScaleHandlers, error) {
	var err error
	promtimer := internalmetrics.GetGoFunctionMetric("business", "ThreeScaleService", "DeleteThreeScaleHandler")
	defer promtimer.ObserveNow(&err)

	conf := config.Get()

	err = in.k8s.DeleteIstioObject(resourceTypesToAPI["adapters"], conf.IstioNamespace, "handlers", handlerName)
	if err != nil {
		return nil, err
	}

	instanceName := "threescale-authorization-" + handlerName
	err = in.k8s.DeleteIstioObject(resourceTypesToAPI["templates"], conf.IstioNamespace, "instances", instanceName)
	if err != nil {
		return nil, err
	}

	return in.getThreeScaleHandlers()
}

func (in *ThreeScaleService) GetThreeScaleRule(namespace, service string) (models.ThreeScaleServiceRule, error) {
	var err error
	promtimer := internalmetrics.GetGoFunctionMetric("business", "ThreeScaleService", "GetThreeScaleRule")
	defer promtimer.ObserveNow(&err)

	return models.ThreeScaleServiceRule{}, nil
}

func generateMatch(threeScaleServiceRule models.ThreeScaleServiceRule) string {
	conf := config.Get()
	match := "destination.service.namespace == \"" + threeScaleServiceRule.ServiceNamespace + "\" && "
	match += "destination.service.name == \"" + threeScaleServiceRule.ServiceName + "\" && "
	match += "destination.labels[\"" + conf.IstioLabels.AppLabelName + "\"] == \"" + threeScaleServiceRule.AppName + "\" && "
	if len(threeScaleServiceRule.Versions) > 0 {
		match += "("
		for i, version := range threeScaleServiceRule.Versions {
			if i > 0 {
				match += "|| "
			}
			match += "destination.labels[\"" + conf.IstioLabels.VersionLabelName + "\"] == \"" + version + "\" "
		}
		match += ")"
	}
	return match
}

func generateJsonRule(threeScaleServiceRule models.ThreeScaleServiceRule) (string, error) {
	conf := config.Get()
	newRule := kubernetes.GenericIstioObject{
		TypeMeta: meta_v1.TypeMeta{
			APIVersion: "config.istio.io/v1alpha2",
			Kind:       "rule",
		},
		ObjectMeta: meta_v1.ObjectMeta{
			Name:      "threescale-" + threeScaleServiceRule.ServiceNamespace + "-" + threeScaleServiceRule.ServiceName,
			Namespace: conf.IstioNamespace,
			Labels: map[string]string{
				"kiali_wizard": threeScaleServiceRule.ServiceNamespace + "-" + threeScaleServiceRule.ServiceName,
			},
		},
		Spec: map[string]interface{}{
			"match": generateMatch(threeScaleServiceRule),
			"actions": []interface{}{
				map[string]interface{}{
					"handler": threeScaleServiceRule.ThreeScaleHandlerName + "." + conf.IstioNamespace,
					"instances": []interface{}{
						"threescale-authorization-" + threeScaleServiceRule.ThreeScaleHandlerName,
					},
				},
			},
		},
	}

	bRule, err := json.Marshal(newRule)
	if err != nil {
		return "", err
	}

	return string(bRule), nil
}

func (in *ThreeScaleService) CreateThreeScaleRule(namespace string, body []byte) (models.ThreeScaleServiceRule, error) {
	var err error
	promtimer := internalmetrics.GetGoFunctionMetric("business", "ThreeScaleService", "CreateThreeScaleRule")
	defer promtimer.ObserveNow(&err)

	threeScaleServiceRule := &models.ThreeScaleServiceRule{}
	err2 := json.Unmarshal(body, threeScaleServiceRule)
	if err2 != nil {
		log.Errorf("JSON: %s shows error: %s", string(body), err2)
		err = fmt.Errorf(models.BadThreeScaleRuleJson)
		return models.ThreeScaleServiceRule{}, err
	}

	jsonRule, err2 := generateJsonRule(*threeScaleServiceRule)
	if err2 != nil {
		log.Error(err2)
		err = fmt.Errorf(models.BadThreeScaleRuleJson)
		return models.ThreeScaleServiceRule{}, err
	}

	conf := config.Get()
	_, errRule := in.k8s.CreateIstioObject(resourceTypesToAPI["rules"], conf.IstioNamespace, "rules", jsonRule)
	if errRule != nil {
		return models.ThreeScaleServiceRule{}, errRule
	}

	return *threeScaleServiceRule, nil
}

func (in *ThreeScaleService) UpdateThreeScaleRule(namespace, service string, body []byte) (models.ThreeScaleServiceRule, error) {
	var err error
	promtimer := internalmetrics.GetGoFunctionMetric("business", "ThreeScaleService", "UpdateThreeScaleRule")
	defer promtimer.ObserveNow(&err)

	threeScaleServiceRule := &models.ThreeScaleServiceRule{}
	err2 := json.Unmarshal(body, threeScaleServiceRule)
	if err2 != nil {
		log.Errorf("JSON: %s shows error: %s", string(body), err2)
		err = fmt.Errorf(models.BadThreeScaleRuleJson)
		return models.ThreeScaleServiceRule{}, err
	}

	// Be sure that parameters are used in the rule
	(*threeScaleServiceRule).ServiceNamespace = namespace
	(*threeScaleServiceRule).ServiceName = service

	jsonRule, err2 := generateJsonRule(*threeScaleServiceRule)
	if err2 != nil {
		log.Error(err2)
		err = fmt.Errorf(models.BadThreeScaleRuleJson)
		return models.ThreeScaleServiceRule{}, err
	}

	ruleName := "threescale-" + namespace + "-" + service

	conf := config.Get()
	_, errRule := in.k8s.UpdateIstioObject(resourceTypesToAPI["rules"], conf.IstioNamespace, "rules", ruleName, jsonRule)
	if errRule != nil {
		return models.ThreeScaleServiceRule{}, errRule
	}

	return *threeScaleServiceRule, nil
}

func (in *ThreeScaleService) DeleteThreeScaleRule(namespace, service string) error {
	var err error
	promtimer := internalmetrics.GetGoFunctionMetric("business", "ThreeScaleService", "DeleteThreeScaleRule")
	defer promtimer.ObserveNow(&err)

	conf := config.Get()
	ruleName := "threescale-" + namespace + "-" + service
	err = in.k8s.DeleteIstioObject(resourceTypesToAPI["rules"], conf.IstioNamespace, "rules", ruleName)
	return err
}
