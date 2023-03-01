/*
Copyright 2023 Mihai Sarbulescu.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controllers

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	subzer0ms "subzer0ms.com/subzero-controller/api/v1beta1"
)

// SubZer0MSReconciler reconciles a SubZer0MS object
type SubZer0MSReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=ship.subzer0ms.com,resources=subzer0ms,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=ship.subzer0ms.com,resources=subzer0ms/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=ship.subzer0ms.com,resources=subzer0ms/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the SubZer0MS object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.14.1/pkg/reconcile
func (r *SubZer0MSReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	_ = log.FromContext(ctx)

	// TODO(user): your logic here

	var input subzer0ms.SubZer0MS
	err := r.Get(ctx, req.NamespacedName, &input)
	if err != nil {
		log.Log.Error(err, "could not get the input")
		return ctrl.Result{}, err
	}

	log.Log.Info(fmt.Sprintf("Reconciling for %s and input %s", req.NamespacedName, input.Spec.Expression))

	podName := fmt.Sprintf("subzer0ms-%s", input.Name)

	pod := corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      podName,
			Namespace: input.Namespace,
		},
		Spec: corev1.PodSpec{
			RestartPolicy: "Never",
			Containers: []corev1.Container{
				{
					Name:  "subzer0ms",
					Image: "python:latest",
					Args:  []string{"python", "-c", fmt.Sprintf("print(%s)", input.Spec.Expression)},
				},
			},
		},
	}

	log.Log.Info(fmt.Sprintf("Creating pod %s", pod.Name))

	err = r.Delete(ctx, &pod, &client.DeleteOptions{})
	if err != nil {
		log.Log.Error(err, "could not delete the pod (ignore if it does not exist)")
	}

	err = r.Create(ctx, &pod, &client.CreateOptions{})
	if err != nil {
		log.Log.Error(err, "could not create the pod")
		return ctrl.Result{}, err
	}

	log.Log.Info(fmt.Sprintf("Created pod %s", pod.Name))

	// wait for the pod to be ready (replace with a watch)
	time.Sleep(10 * time.Second)

	log.Log.Info(fmt.Sprintf("Reading logs from pod %s", pod.Name))

	logs, err := r.ReadPodLogs(ctx, pod)
	if err != nil {
		log.Log.Error(err, "could not read the logs")
		return ctrl.Result{}, err
	}

	log.Log.Info(fmt.Sprintf("Read logs from pod %s", pod.Name))
	log.Log.Info(fmt.Sprintf("Logs: %s", logs))

	input.Status.Answer = logs

	err = r.Update(ctx, &input, &client.UpdateOptions{})
	if err != nil {
		log.Log.Error(err, "could not update with the answer")
		return ctrl.Result{}, err
	}

	log.Log.Info(fmt.Sprintf("Updated with the answer %s", input.Status.Answer))

	return ctrl.Result{}, nil
}

// reads the logs from a pod
func (r *SubZer0MSReconciler) ReadPodLogs(ctx context.Context, pod corev1.Pod) (string, error) {
	podLogOpts := corev1.PodLogOptions{}
	config := ctrl.GetConfigOrDie()

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return "error in getting access to K8S", err
	}

	req := clientset.CoreV1().Pods(pod.Namespace).GetLogs(pod.Name, &podLogOpts)
	podLogs, err := req.Stream(ctx)
	if err != nil {
		log.Log.Error(err, "could not read the logs")
		return "", err
	}

	defer podLogs.Close()

	// can also use ioutil.ReadAll(podLogs) to read the logs
	buf := new(bytes.Buffer)
	_, err = io.Copy(buf, podLogs)
	if err != nil {
		log.Log.Error(err, "could not copy the logs")
		return "", err
	}

	return buf.String(), nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *SubZer0MSReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&subzer0ms.SubZer0MS{}).
		Complete(r)
}
