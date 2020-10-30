package controllers

import (
	"fmt"
	"time"

	nvmeshv1 "excelero.com/nvmesh-k8s-operator/api/v1"
	v1 "k8s.io/api/core/v1"
	fields "k8s.io/apimachinery/pkg/fields"
	"k8s.io/client-go/kubernetes"
	cache "k8s.io/client-go/tools/cache"
)

func (r *NVMeshReconciler) ListenToNodeLabels() (chan struct{}, error) {
	clientset, err := kubernetes.NewForConfig(r.Manager.GetConfig())
	if err != nil {
		return nil, err
	}

	watchlist := cache.NewListWatchFromClient(clientset.CoreV1().RESTClient(), "nodes", "", fields.Everything())
	_, controller := cache.NewInformer(
		watchlist,
		&v1.Node{},
		time.Second*0,
		cache.ResourceEventHandlerFuncs{
			AddFunc: func(obj interface{}) {
				fmt.Printf("node added: %s \n", obj.(*v1.Node).GetName())
			},
			DeleteFunc: func(obj interface{}) {
				fmt.Printf("node deleted: %s \n", obj.(*v1.Node).GetName())
			},
			UpdateFunc: func(oldObj, newObj interface{}) {
				oldNode := oldObj.(*v1.Node)
				newNode := newObj.(*v1.Node)
				r.OnNodeUpdate(oldNode, newNode)
			},
		},
	)

	stopChannel := make(chan struct{})

	go controller.Run(stopChannel)
	return stopChannel, nil
}

func (r *NVMeshReconciler) OnNodeUpdate(oldNode *v1.Node, newNode *v1.Node) {
	var targetRemoved bool
	var clientRemoved bool

	nodeName := newNode.GetName()

	_, clientInNew := newNode.ObjectMeta.Labels[nvmeshClientLabelKey]
	_, clientInOld := oldNode.ObjectMeta.Labels[nvmeshClientLabelKey]

	if !clientInOld && clientInNew {
		fmt.Printf("nvmesh-client added on node %s\n", nodeName)
	}

	if clientInOld && !clientInNew {
		clientRemoved = true
		fmt.Printf("nvmesh-client removed from node %s\n", nodeName)
	}

	_, targetInNew := newNode.ObjectMeta.Labels[nvmeshTargetLabelKey]
	_, targetInOld := oldNode.ObjectMeta.Labels[nvmeshTargetLabelKey]

	if !targetInOld && targetInNew {
		fmt.Printf("nvmesh-target added on node %s\n", nodeName)
	}

	if targetInOld && !targetInNew {
		targetRemoved = true
		fmt.Printf("nvmesh-target removed from node %s\n", nodeName)
	}

	if !targetInNew && !clientInNew && (clientRemoved || targetRemoved) {
		go r.UninstallAndWaitToFinish(nodeName)
	}
}

func (r *NVMeshReconciler) UninstallAndWaitToFinish(nodeName string) {
	namespace := "default"
	jobName := uninstallJobNamePrefix + nodeName

	fakeCR := &nvmeshv1.NVMesh{}
	fakeCR.SetName("uninstall-node-from-event")
	err := r.UninstallNode(fakeCR, nodeName)
	if err != nil {
		fmt.Printf("Failed to create uninstall job on node %s\n", nodeName)
	}

	completed := false
	for !completed {
		result, err := r.waitForJobToFinish(namespace, jobName)
		if !result.Requeue && err == nil {
			break
		}

		time.Sleep(3 * time.Second)
	}

	err = r.deleteJob(namespace, jobName)
	if err != nil {
		fmt.Printf("Failed to delete uninstall job on node %s\n", nodeName)
	}
}
