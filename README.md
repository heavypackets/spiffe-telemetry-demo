# DonutSalon
## Local Setup

First, install the latest `virtualbox`, `minikube` and `kubectl`.

```
minikube start \
    --extra-config=apiserver.Admission.PluginNames="Initializers,NamespaceLifecycle,LimitRanger,ServiceAccount,DefaultStorageClass,GenericAdmissionWebhook,ResourceQuota" \
    --kubernetes-version=v1.7.5
kubectl apply -f istio.yaml
kubectl apply -f istio-initializer.yaml
kubectl apply -n istio-system -f https://raw.githubusercontent.com/jaegertracing/jaeger-kubernetes/master/all-in-one/jaeger-all-in-one-template.yml
export GATEWAY_URL=$(kubectl get po -n istio-system -l istio=ingress -n istio-system -o 'jsonpath={.items[0].status.hostIP}'):$(kubectl get svc istio-ingress -n istio-system -n istio-system -o 'jsonpath={.spec.ports[0].nodePort}')
kubectl port-forward -n istio-system $(kubectl get pod -n istio-system -l app=jaeger -o jsonpath='{.items[0].metadata.name}') 16686:16686 &
```

## Deploy DonutSalon with Istio (Jaeger edition)

```
kubectl create -f instio-app.yaml
```

## Deploy DonutSalon with Docker-Compose (LightStep edition)

```
# run build once, and after any change to app
./build.sh
docker-compose up
```