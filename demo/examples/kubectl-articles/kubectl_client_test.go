package kubectl_articles

import (
	initclient "K8s_demo/demo/examples/client"
	"context"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"log"
	"testing"
	"time"
)

func TestApply(t *testing.T) {
	km := NewKubectlManagerOrDie(initclient.ClientSet.RestConfig)
	if err := km.Apply(context.TODO(), []byte(applyStr)); err != nil {
		log.Fatalf("apply error: %v", err)
	}
}

func TestDelete(t *testing.T) {
	km := NewKubectlManagerOrDie(initclient.ClientSet.RestConfig)
	if err := km.Delete(context.TODO(), []byte(applyStr), true); err != nil {
		log.Fatalf("delete error: %v", err)
	}
}

func TestApplyAndDeleteByFile(t *testing.T) {
	km := NewKubectlManagerOrDie(initclient.ClientSet.RestConfig)
	if err := km.ApplyByFile(context.TODO(), "./test_pod.yaml"); err != nil {
		log.Fatalf("apply error: %v", err)
	}

	if err := km.DeleteByFile(context.TODO(), "./test_pod.yaml", true); err != nil {
		log.Fatalf("delete error: %v", err)
	}
}

func TestApplyAndDeleteByResource(t *testing.T) {

	pod := &corev1.Pod{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "Pod",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "my-pod",
			Namespace: "default",
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Name:  "my-container",
					Image: "nginx",
					Ports: []corev1.ContainerPort{
						{
							ContainerPort: 80,
						},
					},
				},
			},
		},
	}

	km := NewKubectlManagerOrDie(initclient.ClientSet.RestConfig)
	if err := km.ApplyByResource(context.TODO(), pod); err != nil {
		log.Fatalf("apply resource error: %v", err)
	}

	time.Sleep(time.Second * 10)

	if err := km.DeleteByResource(context.TODO(), pod, false); err != nil {
		log.Fatalf("apply resource error: %v", err)
	}

}

const applyStr = `
apiVersion: apps/v1
kind: Deployment
metadata:
  name: nginx
spec:
  replicas: 1
  selector:
    matchLabels:
      app: nginx
  template:
    metadata:
      labels:
        app: nginx
    spec:
      containers:
      - name: nginx
        image: nginx
---
apiVersion: v1
kind: Service
metadata:
  name: nginx-svc
spec:
  ports:
  - name: web
    port: 80
    protocol: TCP
    targetPort: 80
  selector:
    app: nginx
  type: ClusterIP
---
`

const configMapYAML = `
apiVersion: v1
kind: ConfigMap
metadata:
  name: my-configmap
  namespace: default
data:
  key1: value1
  key2: value2
`
