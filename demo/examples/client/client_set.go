package client

import (
	"flag"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"log"
	"os"
	"path/filepath"
)

// 配置文件
func K8sRestConfig() *rest.Config {
	//// 需要注意这里的config文件目录。
	//config, err := clientcmd.BuildConfigFromFlags("", "config")
	//if err != nil {
	//	log.Fatal(err)
	//}

	var kubeConfig *string

	if home := HomeDir(); home != "" {
		kubeConfig = flag.String("kubeconfig", filepath.Join(home, ".kube", "config"), "")
	} else {
		kubeConfig = flag.String("kubeconfig", "", "")
	}
	//flag.Parse()

	// 首先使用 inCluster 模式(需要去配置对应的 RBAC 权限，默认的sa是default->是没有获取deployments的List权限)
	var config *rest.Config
	config, err := rest.InClusterConfig()
	if err != nil {
		// 使用 KubeConfig 文件创建集群配置 Config 对象
		if config, err = clientcmd.BuildConfigFromFlags("", *kubeConfig); err != nil {
			log.Fatal(err)
		}
	}

	return config
}

// 返回初始化k8s-client
func InitClient(config *rest.Config) kubernetes.Interface {
	c, err := kubernetes.NewForConfig(config)
	if err != nil {
		log.Fatal(err)
	}

	return c
}

// 返回初始化k8s-dynamic-client
func InitDynamicClient(config *rest.Config) dynamic.Interface {
	c, err := dynamic.NewForConfig(config)
	if err != nil {
		log.Fatal(err)
	}

	return c
}

func HomeDir() string {
	if h := os.Getenv("HOME"); h != "" {
		return h
	}

	return os.Getenv("USERPROFILE")
}

var ClientSet = &Client{}

type Client struct {
	RestConfig      *rest.Config
	Client          kubernetes.Interface // 因为需要单元测试，所以不要用 *kubernetes.Clientset
	DynamicClient   dynamic.Interface
	DiscoveryClient discovery.DiscoveryInterface
}

func init() {
	config := K8sRestConfig()
	ClientSet.RestConfig = config
	ClientSet.Client = InitClient(config)
	ClientSet.DynamicClient = InitDynamicClient(config)
	ClientSet.DiscoveryClient = InitClient(config).Discovery()
}
