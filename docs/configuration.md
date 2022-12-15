## Configuration

The btp-manager internal settings can be configured using CLI arguments or `ConfigMap`.

To configure them using CLI arguments, follow this example:
```
$ manager --help
Usage of ./manager:
  -chart-path string
    	Path to the root directory inside the chart. (default "./module-chart")
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
  -hard-delete-check-interval duration
    	Hard delete retry interval. (default 10s)
  -secret-name string
    	Secret name with input values for sap-btp-operator chart templating. (default "sap-btp-manager")
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

Configuration with `ConfigMap` [example](../examples/btp-operator-configmap.yaml).
```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: sap-btp-manager
  namespace: kyma-system
data:
  ChartPath: ./module-chart
  ChartNamespace: kyma-system
  SecretName: sap-btp-manager
  DeploymentName: sap-btp-operator-controller-manager
  ProcessingStateRequeueInterval: 5m
  ReadyStateRequeueInterval: 1h
  ReadyTimeout: 1m
  HardDeleteCheckInterval: 10s
```
