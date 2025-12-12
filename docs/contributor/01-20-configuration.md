# Configuration

You can configure BTP Manager using CLI arguments or `ConfigMap`.

To configure the BTP Manager internal settings using CLI arguments, choose the parameters you need, and use them with corresponding custom values:
```
$ manager --help
Usage of ./manager:
  -chart-path string
    	Path to the root directory inside the chart. (default "./module-chart/chart")
  -resources-path string
    Path to the directory with module resources to apply/delete. (default "./module-resources")
  -chart-namespace string
    	Namespace to install chart resources. (default "kyma-system")
  -config-name string
    	ConfigMap name with configuration knobs for the btp-manager internals. (default "sap-btp-manager")
  -deployment-name string
    	Name of the deployment of sap-btp-operator for deprovisioning. (default "sap-btp-operator-controller-manager")
  -hard-delete-timeout duration
    	Hard delete timeout. (default 20m0s)
  -health-probe-bind-address string
    	The address the probe endpoint binds to. (default ":8081")
  -kubeconfig string
    	Paths to a kubeconfig. Only required if out-of-cluster.
  -leader-elect
    	Enable leader election for controller manager. Enabling this will ensure there is only one active controller manager.
  -metrics-bind-address string
    	The address the metric endpoint binds to. (default ":8080")
  -processing-state-requeue-interval duration
    	Requeue interval for state "processing". (default 5m0s)
  -ready-state-requeue-interval duration
    	Requeue interval for state "ready". (default 1h0m0s)
  -ready-timeout duration
    	Helm chart timeout. (default 1m0s)
  -ready-check-interval duration
    	Ready check retry interval. (default 1s)
  -hard-delete-timeout duration
    	Hard delete timeout. (default 20m)
  -hard-delete-check-interval duration
    	Hard delete retry interval. (default 10s)
  -delete-request-timeout duration
    	Delete request timeout in hard delete. (default 5m)
  -enable-limited-cache string
      Enable limited cache for the SAP BTP service operator. When enabled, caches only Secrets and ConfigMaps with the label "services.cloud.sap.com/managed-by-sap-btp-operator: true". (default "false")
  -secret-name string
    	Secret name with input values for sap-btp-operator chart templating. (default "sap-btp-manager")
  -status-update-check-interval duration
        Status update retry interval. (default 500ms)
  -status-update-timeout duration
        Status update timeout. (default 10s)
  -manager-resources-path string
        Path to the directory with BTP Manager resources. (default "./manager-resources")
  -zap-devel
    	Development Mode defaults(encoder=consoleEncoder,logLevel=Debug,stackTraceLevel=Warn). Production Mode defaults(encoder=jsonEncoder,logLevel=Info,stackTraceLevel=Error) (default true)
  -zap-encoder value
    	Zap log encoding (one of 'json' or 'console')
  -zap-log-level value
    	Zap Level to configure the verbosity of logging. Can be one of 'debug', 'info', 'error', or any integer value > 0 which corresponds to custom debug levels of increasing verbosity
  -zap-stacktrace-level value
    	Zap Level at and above which stacktraces are captured (one of 'info', 'error', 'panic').
  -zap-time-encoding value
    	Zap time encoding (one of 'epoch', 'millis', 'nano', 'iso8601', 'rfc3339' or 'rfc3339nano'). Defaults to 'epoch'.
```

To configure BTP Manager with a `ConfigMap`, follow this [example](../../examples/btp-operator-configmap.yaml).  
You should get a result similar to this one:
```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: sap-btp-manager
  namespace: kyma-system
  labels:
    app.kubernetes.io/managed-by: btp-manager
data:
  ChartPath: ./module-chart/chart
  ChartNamespace: kyma-system
  SecretName: sap-btp-manager
  DeploymentName: sap-btp-operator-controller-manager
  ProcessingStateRequeueInterval: 5m
  ReadyStateRequeueInterval: 1h
  ReadyTimeout: 1m
  HardDeleteCheckInterval: 10s
  EnableLimitedCache: false
```
