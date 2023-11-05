package kubectl_articles

import (
	initclient "K8s_demo/demo/examples/client"
	"context"
	"fmt"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"testing"
)

func TestCreatePodByClientGo(t *testing.T) {

	// 构建 Pod 对象的配置
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-pod-by-client-go",
			Namespace: "default",
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Name:  "my-container",
					Image: "nginx:latest",
				},
			},
		},
	}

	// 使用 client-go 创建 Pod
	createdPod, err := initclient.ClientSet.Client.CoreV1().Pods("default").Create(context.TODO(), pod, metav1.CreateOptions{})
	if err != nil {
		panic(err.Error())
	}

	fmt.Printf("Pod created successfully. Name: %s, Namespace: %s\n", createdPod.Name, createdPod.Namespace)
}
