## 浅谈 kubectl apply 与 client-go 之间的区别
目录：
- [1. 背景概述](#t1)
- [2. 比较 kubectl apply 与 client-go 的区别](#t2)
- [3. kubectl apply 源码简析](#t3)
- [4. 牛刀小试！](#t4)

本文基于 kubernetes v1.23 版本，版本有点旧，读者可自行查看更新版本，kubectl 部分源码大同小异

### 1. <a name='t1'></a>背景概述：
背景概述：使用 k8s 的过程中，我们经常会用到 kubernetes 官方提供的 [kubectl](https://github.com/kubernetes/kubectl) 命令行工具，调用方可以很方便的使用类似 kubectl apply xxx 等方式
对集群内资源进行增删改查操作。此外，kubernetes 官方除了提供命令行工具之外，还提供了 [client-go](https://github.com/kubernetes/client-go) 编程库给调用方使用，调用方能够使用 API 的方式在程序中与 k8s api-server交互。
如果读者两种方式都使用过，会不会心中有个疑问？当在使用 kubectl apply xxx 创建资源对象时，跟 client-go 创建是否相同呢？如果不相同，那有什么区别呢？

本文针对这个疑问进行展开，会对比使用 kubectl apply 命令与 client-go 创建资源对象的差别，并对源码进行简要的解析。

### 2. <a name='t2'></a>比较 kubectl apply 与 client-go 的区别：
我们可以先简易比较两者间有哪些差距，操作与代码可参考 [参考代码](../demo/examples/kubectl-articles)
- kubectl apply
```bash
➜  k8s-club git:(main) ✗ kubectl apply -f demo/examples/kubectl-articles/test_pod.yaml
pod/test-pod-by-kubectl created
```
- client-go
```go
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
```
接下来使用 kubectl 命令查看两者的差别。
```bash
➜  k8s-club git:(main) ✗ kubectl get pods test-pod-by-kubectl -oyaml
apiVersion: v1
kind: Pod
metadata:
  annotations:
    kubectl.kubernetes.io/last-applied-configuration: |
      {"apiVersion":"v1","kind":"Pod","metadata":{"annotations":{},"name":"test-pod-by-kubectl","namespace":"default"},"spec":{"containers":[{"image":"nginx:latest","imagePullPolicy":"IfNotPresent","name":"my-container"}]}}
  creationTimestamp: "2023-11-04T11:31:07Z"
  name: test-pod-by-kubectl
  namespace: default
  resourceVersion: "5112730"
  uid: b0097ba1-24be-4360-861a-7d173b0f7833
...
```

```bash
➜  k8s-club git:(main) ✗ kubectl get pods test-pod-by-client-go -oyaml
apiVersion: v1
kind: Pod
metadata:
  creationTimestamp: "2023-11-04T12:39:24Z"
  name: test-pod-by-client-go
  namespace: default
  resourceVersion: "5118126"
  uid: c2916676-e51a-41d0-b470-9ee59a2dfdb1
```
从上面命令可以看出两者之间的区别在 annotations 字段上，使用 kubectl apply 创建时，会多出 "kubectl.kubernetes.io/last-applied-configuration" 的字段，表示最近一次通过 kubectl apply 更新的资源对象，用于对比该次与上次 spec 字段的差距，并进行 patch 操作。

### 3. <a name='t2'></a>kubectl apply 源码简析：

#### 3-1. 重要字段


#### 3-2. 启动流程
从项目入口开始浏览代码，可以发现 kubectl 同样使用 cobra 封装了启动程序。
```go
// staging/src/k8s.io/kubectl/pkg/cmd/cmd.go
// NewDefaultKubectlCommand creates the `kubectl` command with default arguments
func NewDefaultKubectlCommand() *cobra.Command {
	return NewDefaultKubectlCommandWithArgs(KubectlOptions{
		PluginHandler: NewDefaultPluginHandler(plugin.ValidPluginFilenamePrefixes),
		Arguments:     os.Args,
		ConfigFlags:   defaultConfigFlags,
		IOStreams:     genericclioptions.IOStreams{In: os.Stdin, Out: os.Stdout, ErrOut: os.Stderr},
	})
}
```
NewDefaultKubectlCommandWithArgs()方法返回 *cobra.Command 对象，用于执行整个 kubectl 程序。
- 调用 NewKubectlCommand() 方法
- 传入命令行参数，进行查找
```go
// staging/src/k8s.io/kubectl/pkg/cmd/cmd.go
// NewDefaultKubectlCommandWithArgs 创建 kubectl 命令
func NewDefaultKubectlCommandWithArgs(o KubectlOptions) *cobra.Command {
	// 创建 kubectl 
	cmd := NewKubectlCommand(o)
	...

	if len(o.Arguments) > 1 {
		// 这里为传入的参数，即 create -f nginx_pod.yaml 部分
		cmdPathPieces := o.Arguments[1:]
		
		// 调用cobra的Find去匹配args.
		if _, _, err := cmd.Find(cmdPathPieces); err != nil {
			...
		}
	}

	return cmd
}
```
NewKubectlCommand(): 
- 初始化 Factory 接口对象，此对象本质上是一个 Rest Client，用于和 api-server 通信或验证使用，k8s 把其封装成 Factory 对象。
- groups 变量：我们可以发现很多常见的 kubectl 命令都在这里进行初始化与分类(使用 kubectl -h 可以看到分类项)，本文会关注在 **apply** 命令，读者也能自行阅读其他命令。
```go
// staging/src/k8s.io/kubectl/pkg/cmd/cmd.go
// NewKubectlCommand creates the `kubectl` command and its nested children.
func NewKubectlCommand(o KubectlOptions) *cobra.Command {
	warningHandler := rest.NewWarningWriter(o.IOStreams.ErrOut, rest.WarningWriterOptions{Deduplicate: true, Color: term.AllowsColorOutput(o.IOStreams.ErrOut)})
	warningsAsErrors := false
	// Parent command to which all subcommands are added.
	cmds := &cobra.Command{
		Use:   "kubectl",
		Short: i18n.T("kubectl controls the Kubernetes cluster manager"),
		...
		Run: runHelp,
		// Hook before and after Run initialize and write profiles to disk,
		// respectively.
		PersistentPreRunE: 
			...
		},
		PersistentPostRunE: 
			...
		},
	}
	// 准备 opts 与 flags 相关的操作(创建对象、赋值等)
	...

	// 重要的 factory 实例对象
	f := cmdutil.NewFactory(matchVersionKubeConfigFlags)

	...

	// Avoid import cycle by setting ValidArgsFunction here instead of in NewCmdGet()
	getCmd := get.NewCmdGet("kubectl", f, o.IOStreams)
	getCmd.ValidArgsFunction = utilcomp.ResourceTypeAndNameCompletionFunc(f)

	// 由这里可以查看到不同的 kubectl 命令，ex: create get apply patch ...
	groups := templates.CommandGroups{
		{
			// 1. 初级命令，包括 create/expose/run/set
			Message: "Basic Commands (Beginner):",
			Commands: []*cobra.Command{
				create.NewCmdCreate(f, o.IOStreams),
				expose.NewCmdExposeService(f, o.IOStreams),
				run.NewCmdRun(f, o.IOStreams),
				set.NewCmdSet(f, o.IOStreams),
			},
		},
		{
			// 2. 中级命令，包括explain/get/edit/delete
			Message: "Basic Commands (Intermediate):",
			Commands: []*cobra.Command{
				explain.NewCmdExplain("kubectl", f, o.IOStreams),
				getCmd,
				edit.NewCmdEdit(f, o.IOStreams),
				delete.NewCmdDelete(f, o.IOStreams),
			},
		},
		{
			// 3. 部署命令，包括 rollout/scale/autoscale
			Message: "Deploy Commands:",
			Commands: []*cobra.Command{
				rollout.NewCmdRollout(f, o.IOStreams),
				scale.NewCmdScale(f, o.IOStreams),
				autoscale.NewCmdAutoscale(f, o.IOStreams),
			},
		},
		{
			// 4. 集群管理命令，包括 cerfificate/cluster-info/top/cordon/drain/taint
			Message: "Cluster Management Commands:",
			Commands: []*cobra.Command{
				certificates.NewCmdCertificate(f, o.IOStreams),
				clusterinfo.NewCmdClusterInfo(f, o.IOStreams),
				top.NewCmdTop(f, o.IOStreams),
				drain.NewCmdCordon(f, o.IOStreams),
				drain.NewCmdUncordon(f, o.IOStreams),
				drain.NewCmdDrain(f, o.IOStreams),
				taint.NewCmdTaint(f, o.IOStreams),
			},
		},
		{
			// 5. 故障排查和调试，包括 describe/logs/attach/exec/port-forward/proxy/cp/auth
			Message: "Troubleshooting and Debugging Commands:",
			Commands: []*cobra.Command{
				describe.NewCmdDescribe("kubectl", f, o.IOStreams),
				logs.NewCmdLogs(f, o.IOStreams),
				attach.NewCmdAttach(f, o.IOStreams),
				cmdexec.NewCmdExec(f, o.IOStreams),
				portforward.NewCmdPortForward(f, o.IOStreams),
				proxyCmd,
				cp.NewCmdCp(f, o.IOStreams),
				auth.NewCmdAuth(f, o.IOStreams),
				debug.NewCmdDebug(f, o.IOStreams),
				events.NewCmdEvents(f, o.IOStreams),
			},
		},
		{
			// 6. 高级命令，包括diff/apply/patch/replace/wait/convert/kustomize
			Message: "Advanced Commands:",
			Commands: []*cobra.Command{
				diff.NewCmdDiff(f, o.IOStreams),
				apply.NewCmdApply("kubectl", f, o.IOStreams),
				patch.NewCmdPatch(f, o.IOStreams),
				replace.NewCmdReplace(f, o.IOStreams),
				wait.NewCmdWait(f, o.IOStreams),
				kustomize.NewCmdKustomize(o.IOStreams),
			},
		},
		{
			// 7. 设置命令，包括label，annotate，completion
			Message: "Settings Commands:",
			Commands: []*cobra.Command{
				label.NewCmdLabel(f, o.IOStreams),
				annotate.NewCmdAnnotate("kubectl", f, o.IOStreams),
				completion.NewCmdCompletion(o.IOStreams.Out, ""),
			},
		},
	}
	groups.Add(cmds)

	...
    
	// 添加一些不在默认分组内的方法
	cmds.AddCommand(alpha)
	...
	return cmds
}
```
NewCmdApply():
- 初始化 Options 对象，kubectl 当中所有命令，都会初始化各自的 Options 对象，ex: ApplyOptions DeleteOptions CreateOptions 等
- 验证相关逻辑
- 执行 Run() 方法
```go
// NewCmdApply creates the `apply` command
func NewCmdApply(baseName string, f cmdutil.Factory, ioStreams genericclioptions.IOStreams) *cobra.Command {
	flags := NewApplyFlags(ioStreams)

	cmd := &cobra.Command{
		Use:                   "apply (-f FILENAME | -k DIRECTORY)",
		DisableFlagsInUseLine: true,
		Short:                 i18n.T("Apply a configuration to a resource by file name or stdin"),
		Long:                  applyLong,
		Example:               applyExample,
		Run: func(cmd *cobra.Command, args []string) {
			// ApplyOptions 对象
			o, err := flags.ToOptions(f, cmd, baseName, args)
			// 验证相关...
			cmdutil.CheckErr(err)
			cmdutil.CheckErr(o.Validate())
			// 执行逻辑
			cmdutil.CheckErr(o.Run())
		},
	}

	flags.AddFlags(cmd)

	// apply subcommands
	// 子命令
	cmd.AddCommand(NewCmdApplyViewLastApplied(f, flags.IOStreams))
	...
	return cmd
}
```
Run():
- 执行预处理 func
- 获取资源对象
```go
func (o *ApplyOptions) Run() error {
	// 处理预处理 func
	if o.PreProcessorFn != nil {
		klog.V(4).Infof("Running apply pre-processor function")
		if err := o.PreProcessorFn(); err != nil {
			return err
		}
	}

	...

	// Generates the objects using the resource builder if they have not
	// already been stored by calling "SetObjects()" in the pre-processor.
	errs := []error{}
	// 获取资源对象
	infos, err := o.GetObjects()
	if err != nil {
		errs = append(errs, err)
	}
	if len(infos) == 0 && len(errs) == 0 {
		return fmt.Errorf("no objects passed to apply")
	}
	// Iterate through all objects, applying each one.
	// 遍历所有资源对象，执行 applyOneObject 方法
	for _, info := range infos {
		if err := o.applyOneObject(info); err != nil {
			errs = append(errs, err)
		}
	}

	// 处理错误相关
	...

	// 执行后处理 func
	if o.PostProcessorFn != nil {
		klog.V(4).Infof("Running apply post-processor function")
		if err := o.PostProcessorFn(); err != nil {
			return err
		}
	}

	return nil
}
```

```go

func (o *ApplyOptions) GetObjects() ([]*resource.Info, error) {
	var err error = nil
	if !o.objectsCached {
		// 使用创建者模式实现
		r := o.Builder.
			Unstructured().
			Schema(o.Validator).
			ContinueOnError().
			NamespaceParam(o.Namespace).DefaultNamespace().
			// 关键就是这个 FilenameParam
			FilenameParam(o.EnforceNamespace, &o.DeleteOptions.FilenameOptions).
			LabelSelectorParam(o.Selector).
			Flatten().
			Do()
		o.objects, err = r.Infos()
		o.objectsCached = true
	}
	return o.objects, err
}
```