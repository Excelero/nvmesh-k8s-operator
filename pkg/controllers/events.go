package controllers

import (
	nvmeshv1 "excelero.com/nvmesh-k8s-operator/pkg/api/v1"
	v1 "k8s.io/api/core/v1"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	typedcorev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/record"
)

//EventManager - allows to create events on objects
type EventManager struct {
	config   *rest.Config
	recorder record.EventRecorder
}

//Normal - create an event with type Normal
func (e *EventManager) Normal(cr *nvmeshv1.NVMesh, reason string, message string) {
	e.recorder.Event(cr, "Normal", reason, message)
}

//Warning - create an event with type Warning
func (e *EventManager) Warning(cr *nvmeshv1.NVMesh, reason string, message string) {
	e.recorder.Event(cr, "Warning", reason, message)
}

//NewEventManager - create a new EventManager to update events on objects
func NewEventManager(config *rest.Config) (*EventManager, error) {
	recorder, err := getEventRecorder(config)
	if err != nil {
		return nil, err
	}

	e := &EventManager{
		config:   config,
		recorder: recorder,
	}

	return e, nil

}

func getEventRecorder(config *rest.Config) (record.EventRecorder, error) {
	kubeClient, err := typedcorev1.NewForConfig(config)
	if err != nil {
		return nil, err
	}
	eventBroadcaster := record.NewBroadcaster()
	//eventBroadcaster.StartLogging(glog.Infof)
	eventBroadcaster.StartRecordingToSink(&typedcorev1.EventSinkImpl{Interface: kubeClient.Events("")})
	recorder := eventBroadcaster.NewRecorder(clientgoscheme.Scheme, v1.EventSource{Component: "NVMesh Operator"})
	return recorder, nil
}
