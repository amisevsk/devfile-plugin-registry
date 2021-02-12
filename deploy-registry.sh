#!/bin/bash

NAMESPACE=${NAMESPACE:-"devworkspace-plugins"}
echo "Deploying plugin registry to the $NAMESPACE namespace"

if [ "$(kubectl api-resources --api-group='route.openshift.io' | grep -o routes)" == "routes" ]; then
  oc new-project $NAMESPACE || true
  kubectl apply -f deploy/route.yaml -n $NAMESPACE
elif minikube status &>/dev/null; then
  export ROUTING_SUFFIX="$(minikube ip).nip.io"
  envsubst < deploy/k8s/ingress.yaml | kubectl apply -n $NAMESPACE -f -
else
  if [ -n $ROUTING_SUFFIX ]; then
    echo "Environment variable ROUTING_SUFFIX must be defined"
    exit 1
  fi
  envsubst < deploy/k8s/ingress.yaml | kubectl apply -n $NAMESPACE -f -
fi
kubectl apply -n $NAMESPACE -f deploy
