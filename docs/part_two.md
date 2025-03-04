## Part Two: Managing Child Resources and Model Promotion with Ollama

In this section, we will expand on the basics covered in Part One and learn how to create child resources, work with Ollama, and use ReplicaSets to promote models based on a success prompt.

By the end of this part, you will:
- Learn how to create and manage child resources in a Kubernetes operator.
- Understand how to interact with the Ollama API.
- Implement ReplicaSet-based model promotion using a prompt.

Some code/manifest has been provided to you to aid you in this workshop:
  - [MyOllama Helpers](../part-two/api/v1/myollama_helpers.go)
  - [MyOllama Client](../part-two/internal/controller/myollama_client.go)
  - [Sample Resource](../part-two/config/samples/talentarena_v1_myollama.yaml)

### Deploying the Operator in the Kubernetes Cluster

To enable the Ollama prompt API call to function properly, we need to deploy the operator inside the Kubernetes cluster.

Run the following commands to build and deploy the latest operator code in your local Kind cluster:

```sh
export IMG="controller:v$(date +"%s")"
make docker-build
kind load docker-image $IMG        
make deploy
```

This ensures that the latest version of the operator is running inside your cluster.


## Observing changes

Check for a new controller deployment:
```
> watch kubectl get pod -A --selector control-plane=controller-manager 
```
Check controller pod logs:
```
kubectl logs -n <namespace> <pod name> -f
```


### Implementing the Reconcile Function

The `Reconcile` function is the core of our controller. It will be responsible for:
1. Fetching the `MyOllama` resource.
2. Creating and managing a service, used for routing traffic to pods.
3. Managing ReplicaSets based on the resource specification.
4. Calling the Ollama API to verify model success prompt.
5. Ensuring only the valid ReplicaSet is running.

#### Fetching the MyOllama Resource

The first step in the reconcile loop is to retrieve the `MyOllama` resource from the cluster:

```go
obj := &talentarenav1.MyOllama{}
if err := r.Get(ctx, req.NamespacedName, obj); err != nil {
    if k8serrors.IsNotFound(err) {
        log.Info("MyOllama resource not found. Ignoring since object must be deleted")
        return ctrl.Result{}, nil
    }
    log.Error(err, "unable to fetch MyOllama")
    return ctrl.Result{}, client.IgnoreNotFound(err)
}
```

#### Creating a Service

Each `MyOllama` resource requires a corresponding service. We first ensure that the service exists:

```go
svc, err := obj.ChildService(r.Scheme))
if err != nil {
    return ctrl.Result{}, err
}
// TODO: create/update service using the k8s client; r.Create or r.Update func.
// doc: https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.20.2/pkg/client
```

#### Managing ReplicaSets

We need to check if a valid ReplicaSet exists for the given `MyOllama` resource.
Note that:
  - all child ReplicaSets created have the `"ollama-ref"=obj.Name` label set.
  - the current expected replicaset has the `"ollama-hash"=obj.Hash()` label.

With this information we can list all children created:
```go
var rsList appsv1.ReplicaSetList
if err := r.List(ctx, &rsList, &client.ListOptions{
    Namespace:     req.Namespace,
    LabelSelector: labels.SelectorFromSet(labels.Set{"ollama-ref": obj.Name}),
}); err != nil {
    log.Error(err, "Failed to list ReplicaSets")
    return ctrl.Result{}, err
}
```

We can then determine which ReplicaSets are expected and which should be removed:

```go
var expected *appsv1.ReplicaSet
var old []appsv1.ReplicaSet

for _, rs := range rsList.Items {
    if rs.Labels["ollama-hash"] == obj.Hash() {
        expected = &rs
    } else {
        old = append(old, rs)
    }
}
```

If no expected ReplicaSet exists, we should create a new one:

```go
rs, err := obj.ChildReplicaSet()
if err != nil {
	return ctrl.Result{}, err
}
// TODO create new ReplicaSet using the k8s client
```

We should wait until the expected pods are created for the expected replica set:
```go
if expected.Status.ReadyReplicas == 0 {
	log.Info("ReplicaSet is not ready", "name", expected.Name)
	return ctrl.Result{RequeueAfter: time.Minute}, nil
}
```

#### Calling the Ollama API

Once a valid ReplicaSet is running, we check if it meets the success prompt criteria by calling the Ollama API:

```go
addr := fmt.Sprintf("%s.%s.svc.cluster.local:11434", svc.Name, svc.Namespace)
if ok, err := checkMyModel(ctx, addr, obj.Spec.Model, obj.Spec.SuccessPrompt); err != nil {
    log.Error(err, "Failed to call Ollama API")
    return ctrl.Result{}, err
} else if !ok {
    return ctrl.Result{RequeueAfter: time.Minute}, nil
}
```

#### Cleaning Up Old ReplicaSets

Finally, we remove any outdated ReplicaSets:

```go
for _, rs := range old {
    if err := r.Delete(ctx, &rs); err != nil {
        log.Error(err, "Failed to delete ReplicaSet")
        return ctrl.Result{}, err
    }
}
```

### Summary

In this part, we extended our operator to:
1. Ensure a corresponding service exists for each `MyOllama` resource.
2. Manage ReplicaSets based on the resource specification.
3. Call the Ollama API to verify model readiness.
4. Clean up outdated ReplicaSets.


Congratulations! You have now implemented a Kubernetes operator that can deploy and promote models based on prompts 🎉


### See it in action

You can use the sample `kubectl apply -f ./config/samples/talentarena_v1_myollama.yaml` to see this in action.
Promoting `smollm:135m` to `llama3.2:1b` and vice versa should give you interesting findings.

You can also try it out locally on your machine by port-forwarding:
```
kubectl port-forward service/<your service name> 11434:11434
```
Then try calling the api:
```
 curl http://localhost:11434/api/generate -d '{
  "model": "llama3.2:1b",
  "prompt":"Is the sky blue?",
  "stream": false
}'
```