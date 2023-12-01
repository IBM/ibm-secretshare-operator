# ibm-secretshare-operator
IBM SecretShare Operator is an Operator to Share Secrets and ConfigMaps between namespaces.

The SecretShare operator and accompanying Custom Resource watches secrets and config maps in a namespace, copying the ones specified in the SecretShare CR to other namespaces, and optionally, to other names in those namespaces.

## SecretShare Custom Resource

An example SecretShare CR specification is shown below:

```
apiVersion: ibmcpcs.ibm.com/v1
kind: SecretShare
metadata:
  name: common-services
  namespace: ibm-common-services
spec:
  # Secrets to share for adopter compatibility to Common Services 3.2.4
  secretshares:
  - secretname: icp-management-ingress-tls-secret
    sharewith:
    - namespace: kube-system
  - secretname: grafana-secret
    sharewith:
    - namespace: kube-system
      name: monitoring-grafana-secret
  - secretname: icp-metering-api-secret
    sharewith:
    - namespace: kube-system
  - secretname: oauth-client-secret
    sharewith:
    - namespace: kube-public
  - secretname: ibmcloud-cluster-ca-cert
    sharewith:
    - namespace: kube-public
  # ConfigMaps to share for adopter compatibility to Common Services 3.2.4
  configmapshares: 
  - configmapname: oauth-client-map
    sharewith:
    - namespace: kube-public
  - configmapname: ibm-cloud-info
    sharewith:
    - namespace: kube-system
  - configmapname: ibmcloud-cluster-info
    sharewith:
    - namespace: kube-public
  - configmapname: common-web-ui-config
    sharewith:
    - namespace: kube-system
  - configmapname: common-web-ui-log4js
    sharewith:
    - namespace: kube-system
  
```

In this example, a SecretShare custom resource named *common-services* would be created in the *ibm-common-services* namespace.

The operator watches all secrets and configmaps in the namespace and creates or updates copies (if they are changed) to the namespaces and (optionally) names specified in the CR.  The specification above would cause the following to be done by the operator (this list is not complete, but an example from which you can get the behavior):

1. The secret named *icp-management-ingress-tls-secret*, if/when found, would be copied into namespace *kube-system*.
2. The secret named *grafana-secret*, if/when found, would be copied into namespace *kube-system* with the name *monitoring-grafana-secret*.
3. The secret named *icp-metering-api-secret*, if/when found, would be copied into namespace *kube-system*.
5. The configmap named *ibm-cloud-info*, if/when found, would be copied into namespace *kube-public*.

The operator watches the SecretShare CR, as well as all secrets and configmaps, so changes in any of these cause the CR to re-evaluate and copy changes as needed.

If the target namespace for a copy does not exist, the SecretShare operator will create the namespace before copying the Secret or ConfigMap
