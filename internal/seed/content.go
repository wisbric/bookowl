package seed

// Tiptap JSON content for seed-demo documents.

const podCrashloopContent = `{"type":"doc","content":[
  {"type":"heading","attrs":{"level":1},"content":[{"type":"text","text":"Pod CrashLoopBackOff"}]},
  {"type":"calloutBlock","attrs":{"type":"danger"},"content":[{"type":"paragraph","content":[{"type":"text","text":"Severity: Critical — Pods are failing to start and Kubernetes is restarting them repeatedly."}]}]},
  {"type":"heading","attrs":{"level":2},"content":[{"type":"text","text":"Symptoms"}]},
  {"type":"bulletList","content":[
    {"type":"listItem","content":[{"type":"paragraph","content":[{"type":"text","text":"Pod status shows CrashLoopBackOff"}]}]},
    {"type":"listItem","content":[{"type":"paragraph","content":[{"type":"text","text":"Container restart count keeps incrementing"}]}]},
    {"type":"listItem","content":[{"type":"paragraph","content":[{"type":"text","text":"Application logs show startup failures"}]}]}
  ]},
  {"type":"heading","attrs":{"level":2},"content":[{"type":"text","text":"Investigation Steps"}]},
  {"type":"taskList","content":[
    {"type":"taskItem","attrs":{"checked":false},"content":[{"type":"paragraph","content":[{"type":"text","text":"Check pod events: kubectl describe pod <name> -n <namespace>"}]}]},
    {"type":"taskItem","attrs":{"checked":false},"content":[{"type":"paragraph","content":[{"type":"text","text":"Check container logs: kubectl logs <pod> -n <namespace> --previous"}]}]},
    {"type":"taskItem","attrs":{"checked":false},"content":[{"type":"paragraph","content":[{"type":"text","text":"Verify image pull: check imagePullPolicy and registry access"}]}]},
    {"type":"taskItem","attrs":{"checked":false},"content":[{"type":"paragraph","content":[{"type":"text","text":"Check resource limits: ensure memory/CPU limits are adequate"}]}]},
    {"type":"taskItem","attrs":{"checked":false},"content":[{"type":"paragraph","content":[{"type":"text","text":"Check liveness/readiness probes: verify endpoints respond correctly"}]}]}
  ]},
  {"type":"heading","attrs":{"level":2},"content":[{"type":"text","text":"Common Fixes"}]},
  {"type":"orderedList","content":[
    {"type":"listItem","content":[{"type":"paragraph","content":[{"type":"text","text":"Fix application config errors (missing env vars, bad config files)"}]}]},
    {"type":"listItem","content":[{"type":"paragraph","content":[{"type":"text","text":"Increase resource limits if OOMKilled"}]}]},
    {"type":"listItem","content":[{"type":"paragraph","content":[{"type":"text","text":"Fix image tag or registry credentials if ImagePullBackOff"}]}]},
    {"type":"listItem","content":[{"type":"paragraph","content":[{"type":"text","text":"Adjust probe thresholds if health checks fail during startup"}]}]}
  ]}
]}`

const oomKilledContent = `{"type":"doc","content":[
  {"type":"heading","attrs":{"level":1},"content":[{"type":"text","text":"Container OOMKilled"}]},
  {"type":"calloutBlock","attrs":{"type":"warning"},"content":[{"type":"paragraph","content":[{"type":"text","text":"The container exceeded its memory limit and was killed by the kernel OOM killer."}]}]},
  {"type":"heading","attrs":{"level":2},"content":[{"type":"text","text":"Investigation"}]},
  {"type":"taskList","content":[
    {"type":"taskItem","attrs":{"checked":false},"content":[{"type":"paragraph","content":[{"type":"text","text":"Check exit code (137 = OOMKilled): kubectl describe pod <name>"}]}]},
    {"type":"taskItem","attrs":{"checked":false},"content":[{"type":"paragraph","content":[{"type":"text","text":"Review memory usage: kubectl top pod <name>"}]}]},
    {"type":"taskItem","attrs":{"checked":false},"content":[{"type":"paragraph","content":[{"type":"text","text":"Check Grafana dashboard for memory trends over last 24h"}]}]},
    {"type":"taskItem","attrs":{"checked":false},"content":[{"type":"paragraph","content":[{"type":"text","text":"Check for memory leaks in application logs"}]}]}
  ]},
  {"type":"heading","attrs":{"level":2},"content":[{"type":"text","text":"Resolution"}]},
  {"type":"paragraph","content":[{"type":"text","text":"If the OOM is legitimate (app needs more memory), increase the memory limit in the deployment spec. If it's a leak, roll back to the last known good version and investigate."}]},
  {"type":"codeBlock","attrs":{"language":"yaml"},"content":[{"type":"text","text":"resources:\n  limits:\n    memory: \"512Mi\"  # Increase as needed\n  requests:\n    memory: \"256Mi\""}]}
]}`

const certExpiryContent = `{"type":"doc","content":[
  {"type":"heading","attrs":{"level":1},"content":[{"type":"text","text":"TLS Certificate Expiry"}]},
  {"type":"calloutBlock","attrs":{"type":"warning"},"content":[{"type":"paragraph","content":[{"type":"text","text":"A TLS certificate is expiring within 7 days. Action required to prevent service outage."}]}]},
  {"type":"heading","attrs":{"level":2},"content":[{"type":"text","text":"Check Certificate"}]},
  {"type":"codeBlock","attrs":{"language":"bash"},"content":[{"type":"text","text":"# Check cert expiry\nopenssl s_client -connect <host>:443 -servername <host> 2>/dev/null | openssl x509 -noout -dates\n\n# Check cert-manager certificates\nkubectl get certificates -A\nkubectl describe certificate <name> -n <namespace>"}]},
  {"type":"heading","attrs":{"level":2},"content":[{"type":"text","text":"Resolution"}]},
  {"type":"taskList","content":[
    {"type":"taskItem","attrs":{"checked":false},"content":[{"type":"paragraph","content":[{"type":"text","text":"If cert-manager: check issuer status and trigger manual renewal"}]}]},
    {"type":"taskItem","attrs":{"checked":false},"content":[{"type":"paragraph","content":[{"type":"text","text":"If manual cert: request new cert from CA and update the secret"}]}]},
    {"type":"taskItem","attrs":{"checked":false},"content":[{"type":"paragraph","content":[{"type":"text","text":"Verify renewal: check new expiry date after update"}]}]},
    {"type":"taskItem","attrs":{"checked":false},"content":[{"type":"paragraph","content":[{"type":"text","text":"Confirm services reloaded: check ingress controller logs"}]}]}
  ]}
]}`

const nodeNotReadyContent = `{"type":"doc","content":[
  {"type":"heading","attrs":{"level":1},"content":[{"type":"text","text":"Node Not Ready"}]},
  {"type":"calloutBlock","attrs":{"type":"danger"},"content":[{"type":"paragraph","content":[{"type":"text","text":"A Kubernetes node has entered NotReady state. Pods may be evicted."}]}]},
  {"type":"heading","attrs":{"level":2},"content":[{"type":"text","text":"Investigation"}]},
  {"type":"taskList","content":[
    {"type":"taskItem","attrs":{"checked":false},"content":[{"type":"paragraph","content":[{"type":"text","text":"Check node status: kubectl get nodes"}]}]},
    {"type":"taskItem","attrs":{"checked":false},"content":[{"type":"paragraph","content":[{"type":"text","text":"Check node conditions: kubectl describe node <name>"}]}]},
    {"type":"taskItem","attrs":{"checked":false},"content":[{"type":"paragraph","content":[{"type":"text","text":"Check kubelet logs: journalctl -u kubelet -f on the node"}]}]},
    {"type":"taskItem","attrs":{"checked":false},"content":[{"type":"paragraph","content":[{"type":"text","text":"Check disk pressure, memory pressure, PID pressure conditions"}]}]},
    {"type":"taskItem","attrs":{"checked":false},"content":[{"type":"paragraph","content":[{"type":"text","text":"Check cloud provider console for instance issues"}]}]}
  ]},
  {"type":"heading","attrs":{"level":2},"content":[{"type":"text","text":"Escalation"}]},
  {"type":"paragraph","content":[{"type":"text","text":"If the node cannot be recovered within 15 minutes, cordon it and drain workloads to healthy nodes:"}]},
  {"type":"codeBlock","attrs":{"language":"bash"},"content":[{"type":"text","text":"kubectl cordon <node-name>\nkubectl drain <node-name> --ignore-daemonsets --delete-emptydir-data"}]}
]}`

const pvcStuckContent = `{"type":"doc","content":[
  {"type":"heading","attrs":{"level":1},"content":[{"type":"text","text":"PVC Stuck Pending"}]},
  {"type":"calloutBlock","attrs":{"type":"info"},"content":[{"type":"paragraph","content":[{"type":"text","text":"A PersistentVolumeClaim is stuck in Pending state and the pod cannot start."}]}]},
  {"type":"heading","attrs":{"level":2},"content":[{"type":"text","text":"Investigation"}]},
  {"type":"taskList","content":[
    {"type":"taskItem","attrs":{"checked":false},"content":[{"type":"paragraph","content":[{"type":"text","text":"Check PVC status: kubectl get pvc -n <namespace>"}]}]},
    {"type":"taskItem","attrs":{"checked":false},"content":[{"type":"paragraph","content":[{"type":"text","text":"Check PVC events: kubectl describe pvc <name> -n <namespace>"}]}]},
    {"type":"taskItem","attrs":{"checked":false},"content":[{"type":"paragraph","content":[{"type":"text","text":"Verify StorageClass exists: kubectl get storageclass"}]}]},
    {"type":"taskItem","attrs":{"checked":false},"content":[{"type":"paragraph","content":[{"type":"text","text":"Check CSI driver pods are running: kubectl get pods -n kube-system"}]}]},
    {"type":"taskItem","attrs":{"checked":false},"content":[{"type":"paragraph","content":[{"type":"text","text":"Check cloud provider volume quotas"}]}]}
  ]}
]}`

const etcdLatencyContent = `{"type":"doc","content":[
  {"type":"heading","attrs":{"level":1},"content":[{"type":"text","text":"etcd High Latency"}]},
  {"type":"calloutBlock","attrs":{"type":"danger"},"content":[{"type":"paragraph","content":[{"type":"text","text":"etcd is experiencing high latency. This can cause API server timeouts and cluster instability."}]}]},
  {"type":"heading","attrs":{"level":2},"content":[{"type":"text","text":"Investigation"}]},
  {"type":"taskList","content":[
    {"type":"taskItem","attrs":{"checked":false},"content":[{"type":"paragraph","content":[{"type":"text","text":"Check etcd metrics: disk_wal_fsync_duration, disk_backend_commit_duration"}]}]},
    {"type":"taskItem","attrs":{"checked":false},"content":[{"type":"paragraph","content":[{"type":"text","text":"Check disk I/O on etcd nodes: iostat -x 1"}]}]},
    {"type":"taskItem","attrs":{"checked":false},"content":[{"type":"paragraph","content":[{"type":"text","text":"Check etcd cluster health: etcdctl endpoint health"}]}]},
    {"type":"taskItem","attrs":{"checked":false},"content":[{"type":"paragraph","content":[{"type":"text","text":"Check for large key-value entries: etcdctl check perf"}]}]}
  ]},
  {"type":"heading","attrs":{"level":2},"content":[{"type":"text","text":"Mitigation"}]},
  {"type":"paragraph","content":[{"type":"text","text":"If disk I/O is the bottleneck, consider moving etcd to faster SSD storage. Run a defragmentation if the database has grown significantly:"}]},
  {"type":"codeBlock","attrs":{"language":"bash"},"content":[{"type":"text","text":"etcdctl defrag --endpoints=https://etcd-0:2379,https://etcd-1:2379,https://etcd-2:2379"}]}
]}`

const pmPoolExhaustionContent = `{"type":"doc","content":[
  {"type":"heading","attrs":{"level":1},"content":[{"type":"text","text":"Post-Mortem: Database Connection Pool Exhaustion"}]},
  {"type":"heading","attrs":{"level":2},"content":[{"type":"text","text":"Incident Summary"}]},
  {"type":"paragraph","content":[{"type":"text","text":"On 2025-12-15 at 14:32 UTC, the API service began returning 500 errors. Investigation revealed the PostgreSQL connection pool was exhausted due to a leaked connection in the document search handler. The incident lasted 47 minutes and affected all API users."}]},
  {"type":"heading","attrs":{"level":2},"content":[{"type":"text","text":"Timeline"}]},
  {"type":"bulletList","content":[
    {"type":"listItem","content":[{"type":"paragraph","content":[{"type":"text","text":"14:32 — Alerts fire: API 5xx rate > 5%"}]}]},
    {"type":"listItem","content":[{"type":"paragraph","content":[{"type":"text","text":"14:35 — On-call engineer Stefan K. acknowledges"}]}]},
    {"type":"listItem","content":[{"type":"paragraph","content":[{"type":"text","text":"14:42 — Identified connection pool exhaustion via Grafana"}]}]},
    {"type":"listItem","content":[{"type":"paragraph","content":[{"type":"text","text":"14:55 — Root cause found: missing defer conn.Release() in search handler"}]}]},
    {"type":"listItem","content":[{"type":"paragraph","content":[{"type":"text","text":"15:05 — Fix deployed, connections recovering"}]}]},
    {"type":"listItem","content":[{"type":"paragraph","content":[{"type":"text","text":"15:19 — All clear, monitoring stable for 15 minutes"}]}]}
  ]},
  {"type":"heading","attrs":{"level":2},"content":[{"type":"text","text":"Root Cause"}]},
  {"type":"paragraph","content":[{"type":"text","text":"A connection acquired for full-text search was not released on the error path. Under load, this gradually consumed all available connections."}]},
  {"type":"heading","attrs":{"level":2},"content":[{"type":"text","text":"Action Items"}]},
  {"type":"taskList","content":[
    {"type":"taskItem","attrs":{"checked":true},"content":[{"type":"paragraph","content":[{"type":"text","text":"Add defer conn.Release() immediately after Acquire"}]}]},
    {"type":"taskItem","attrs":{"checked":true},"content":[{"type":"paragraph","content":[{"type":"text","text":"Add connection pool utilization metric to Grafana dashboard"}]}]},
    {"type":"taskItem","attrs":{"checked":false},"content":[{"type":"paragraph","content":[{"type":"text","text":"Add lint rule to detect missing defer after pool.Acquire"}]}]},
    {"type":"taskItem","attrs":{"checked":false},"content":[{"type":"paragraph","content":[{"type":"text","text":"Set connection pool max lifetime to 5 minutes"}]}]}
  ]}
]}`

const pmCDNCacheContent = `{"type":"doc","content":[
  {"type":"heading","attrs":{"level":1},"content":[{"type":"text","text":"Post-Mortem: CDN Cache Invalidation Storm"}]},
  {"type":"heading","attrs":{"level":2},"content":[{"type":"text","text":"Incident Summary"}]},
  {"type":"paragraph","content":[{"type":"text","text":"On 2026-01-08 at 09:15 UTC, a bulk document update triggered cache invalidation for all CDN edge nodes simultaneously. Origin traffic spiked 40x, causing cascading failures across the API tier. Duration: 1 hour 23 minutes."}]},
  {"type":"heading","attrs":{"level":2},"content":[{"type":"text","text":"Impact"}]},
  {"type":"bulletList","content":[
    {"type":"listItem","content":[{"type":"paragraph","content":[{"type":"text","text":"All users experienced slow page loads (10-30s) or timeouts"}]}]},
    {"type":"listItem","content":[{"type":"paragraph","content":[{"type":"text","text":"Document editing was unavailable for ~45 minutes"}]}]},
    {"type":"listItem","content":[{"type":"paragraph","content":[{"type":"text","text":"NightOwl alert detail pages failed to load runbook content"}]}]}
  ]},
  {"type":"heading","attrs":{"level":2},"content":[{"type":"text","text":"Root Cause"}]},
  {"type":"paragraph","content":[{"type":"text","text":"The bulk migration script invalidated CDN cache keys one-by-one in a tight loop without rate limiting. Each invalidation caused all edge nodes to fetch from origin simultaneously."}]},
  {"type":"heading","attrs":{"level":2},"content":[{"type":"text","text":"Action Items"}]},
  {"type":"taskList","content":[
    {"type":"taskItem","attrs":{"checked":true},"content":[{"type":"paragraph","content":[{"type":"text","text":"Add rate limiting to cache invalidation (max 10/second)"}]}]},
    {"type":"taskItem","attrs":{"checked":false},"content":[{"type":"paragraph","content":[{"type":"text","text":"Implement stale-while-revalidate cache strategy"}]}]},
    {"type":"taskItem","attrs":{"checked":false},"content":[{"type":"paragraph","content":[{"type":"text","text":"Add circuit breaker on origin requests"}]}]}
  ]}
]}`

const liveContextDocContent = `{"type":"doc","content":[
  {"type":"heading","attrs":{"level":1},"content":[{"type":"text","text":"Platform Service Overview"}]},
  {"type":"paragraph","content":[{"type":"text","text":"This document provides a live overview of the platform's operational status. The blocks below pull real-time data from NightOwl."}]},
  {"type":"heading","attrs":{"level":2},"content":[{"type":"text","text":"Current On-Call"}]},
  {"type":"liveContextBlock","attrs":{"subtype":"oncall","rosterId":"platform-primary","serviceName":null,"alertId":null,"label":"Platform Team"}},
  {"type":"heading","attrs":{"level":2},"content":[{"type":"text","text":"Service Health"}]},
  {"type":"liveContextBlock","attrs":{"subtype":"service-status","rosterId":null,"serviceName":"bookowl-api","alertId":null,"label":"BookOwl API"}},
  {"type":"liveContextBlock","attrs":{"subtype":"service-status","rosterId":null,"serviceName":"nightowl-api","alertId":null,"label":"NightOwl API"}},
  {"type":"heading","attrs":{"level":2},"content":[{"type":"text","text":"Active Alerts"}]},
  {"type":"liveContextBlock","attrs":{"subtype":"alerts","rosterId":null,"serviceName":null,"alertId":null,"label":"All Active Alerts"}}
]}`

// --- Additional runbooks ---

const dnsResolutionContent = `{"type":"doc","content":[
  {"type":"heading","attrs":{"level":1},"content":[{"type":"text","text":"DNS Resolution Failures"}]},
  {"type":"calloutBlock","attrs":{"type":"danger"},"content":[{"type":"paragraph","content":[{"type":"text","text":"Services are experiencing DNS resolution failures. This can cascade to all service-to-service communication."}]}]},
  {"type":"heading","attrs":{"level":2},"content":[{"type":"text","text":"Symptoms"}]},
  {"type":"bulletList","content":[
    {"type":"listItem","content":[{"type":"paragraph","content":[{"type":"text","text":"Services logging 'dial tcp: lookup <host>: no such host' errors"}]}]},
    {"type":"listItem","content":[{"type":"paragraph","content":[{"type":"text","text":"Intermittent timeouts on HTTP requests between services"}]}]},
    {"type":"listItem","content":[{"type":"paragraph","content":[{"type":"text","text":"CoreDNS pods showing high CPU or memory usage"}]}]},
    {"type":"listItem","content":[{"type":"paragraph","content":[{"type":"text","text":"nslookup / dig commands failing from within pods"}]}]}
  ]},
  {"type":"heading","attrs":{"level":2},"content":[{"type":"text","text":"Investigation Steps"}]},
  {"type":"taskList","content":[
    {"type":"taskItem","attrs":{"checked":false},"content":[{"type":"paragraph","content":[{"type":"text","text":"Check CoreDNS pod status: kubectl get pods -n kube-system -l k8s-app=kube-dns"}]}]},
    {"type":"taskItem","attrs":{"checked":false},"content":[{"type":"paragraph","content":[{"type":"text","text":"Check CoreDNS logs for errors: kubectl logs -n kube-system -l k8s-app=kube-dns --tail=100"}]}]},
    {"type":"taskItem","attrs":{"checked":false},"content":[{"type":"paragraph","content":[{"type":"text","text":"Test DNS from a debug pod: kubectl run -it --rm dns-test --image=busybox -- nslookup kubernetes.default"}]}]},
    {"type":"taskItem","attrs":{"checked":false},"content":[{"type":"paragraph","content":[{"type":"text","text":"Check CoreDNS ConfigMap for misconfigurations: kubectl get configmap coredns -n kube-system -o yaml"}]}]},
    {"type":"taskItem","attrs":{"checked":false},"content":[{"type":"paragraph","content":[{"type":"text","text":"Verify the kube-dns Service has endpoints: kubectl get endpoints kube-dns -n kube-system"}]}]},
    {"type":"taskItem","attrs":{"checked":false},"content":[{"type":"paragraph","content":[{"type":"text","text":"Check ndots configuration in affected pod: cat /etc/resolv.conf inside the pod"}]}]}
  ]},
  {"type":"heading","attrs":{"level":2},"content":[{"type":"text","text":"Common Fixes"}]},
  {"type":"orderedList","content":[
    {"type":"listItem","content":[{"type":"paragraph","content":[{"type":"text","text":"Restart CoreDNS pods if they are in a bad state: kubectl rollout restart deployment/coredns -n kube-system"}]}]},
    {"type":"listItem","content":[{"type":"paragraph","content":[{"type":"text","text":"Scale up CoreDNS if CPU is saturated: kubectl scale deployment/coredns -n kube-system --replicas=4"}]}]},
    {"type":"listItem","content":[{"type":"paragraph","content":[{"type":"text","text":"Reduce search domains by setting ndots:2 in pod specs to avoid excessive DNS queries"}]}]},
    {"type":"listItem","content":[{"type":"paragraph","content":[{"type":"text","text":"Enable DNS caching in applications (e.g., NodeJS: set dns.lookup with { hints: dns.ADDRCONFIG })"}]}]}
  ]},
  {"type":"heading","attrs":{"level":2},"content":[{"type":"text","text":"Prevention"}]},
  {"type":"paragraph","content":[{"type":"text","text":"Set up monitoring for CoreDNS latency and error rates. Create alerts when p99 latency exceeds 200ms or when the SERVFAIL rate exceeds 1%. Consider running NodeLocal DNSCache for large clusters."}]},
  {"type":"codeBlock","attrs":{"language":"yaml"},"content":[{"type":"text","text":"# NodeLocal DNSCache DaemonSet (excerpt)\napiVersion: apps/v1\nkind: DaemonSet\nmetadata:\n  name: node-local-dns\n  namespace: kube-system\nspec:\n  template:\n    spec:\n      containers:\n      - name: node-cache\n        image: registry.k8s.io/dns/k8s-dns-node-cache:1.22.28"}]}
]}`

const highCPUContent = `{"type":"doc","content":[
  {"type":"heading","attrs":{"level":1},"content":[{"type":"text","text":"High CPU Usage Troubleshooting"}]},
  {"type":"calloutBlock","attrs":{"type":"warning"},"content":[{"type":"paragraph","content":[{"type":"text","text":"A service is consuming excessive CPU. This may cause throttling, increased latency, and degraded performance."}]}]},
  {"type":"heading","attrs":{"level":2},"content":[{"type":"text","text":"Symptoms"}]},
  {"type":"bulletList","content":[
    {"type":"listItem","content":[{"type":"paragraph","content":[{"type":"text","text":"Container CPU usage consistently above 80% of its limit"}]}]},
    {"type":"listItem","content":[{"type":"paragraph","content":[{"type":"text","text":"Increased request latency (p99 above SLO threshold)"}]}]},
    {"type":"listItem","content":[{"type":"paragraph","content":[{"type":"text","text":"CPU throttling visible in container metrics (nr_throttled > 0)"}]}]},
    {"type":"listItem","content":[{"type":"paragraph","content":[{"type":"text","text":"HPA scaling events (if configured) triggering frequently"}]}]}
  ]},
  {"type":"heading","attrs":{"level":2},"content":[{"type":"text","text":"Investigation Steps"}]},
  {"type":"taskList","content":[
    {"type":"taskItem","attrs":{"checked":false},"content":[{"type":"paragraph","content":[{"type":"text","text":"Check current CPU usage: kubectl top pods -n <namespace> --sort-by=cpu"}]}]},
    {"type":"taskItem","attrs":{"checked":false},"content":[{"type":"paragraph","content":[{"type":"text","text":"Check CPU limits and requests: kubectl get pod <pod> -o jsonpath='{.spec.containers[0].resources}'"}]}]},
    {"type":"taskItem","attrs":{"checked":false},"content":[{"type":"paragraph","content":[{"type":"text","text":"Check for CPU throttling in Grafana: container_cpu_cfs_throttled_seconds_total"}]}]},
    {"type":"taskItem","attrs":{"checked":false},"content":[{"type":"paragraph","content":[{"type":"text","text":"Check if a recent deployment caused the spike: kubectl rollout history deployment/<name>"}]}]},
    {"type":"taskItem","attrs":{"checked":false},"content":[{"type":"paragraph","content":[{"type":"text","text":"Profile the application if the issue persists (Go: pprof, Java: JFR, Node: clinic)"}]}]}
  ]},
  {"type":"heading","attrs":{"level":2},"content":[{"type":"text","text":"Immediate Actions"}]},
  {"type":"orderedList","content":[
    {"type":"listItem","content":[{"type":"paragraph","content":[{"type":"text","text":"If traffic spike: check if HPA is configured and increase maxReplicas if needed"}]}]},
    {"type":"listItem","content":[{"type":"paragraph","content":[{"type":"text","text":"If code regression: roll back to previous version: kubectl rollout undo deployment/<name>"}]}]},
    {"type":"listItem","content":[{"type":"paragraph","content":[{"type":"text","text":"If sustained: increase CPU limits temporarily while investigating root cause"}]}]}
  ]},
  {"type":"codeBlock","attrs":{"language":"bash"},"content":[{"type":"text","text":"# Quick check: top CPU consumers across all namespaces\nkubectl top pods -A --sort-by=cpu | head -20\n\n# Check throttling for a specific pod\nkubectl exec <pod> -- cat /sys/fs/cgroup/cpu/cpu.stat"}]}
]}`

const ingressTroubleshootContent = `{"type":"doc","content":[
  {"type":"heading","attrs":{"level":1},"content":[{"type":"text","text":"Ingress Troubleshooting Guide"}]},
  {"type":"paragraph","content":[{"type":"text","text":"This runbook covers common ingress issues including 502/503/504 errors, SSL termination failures, and routing misconfigurations."}]},
  {"type":"heading","attrs":{"level":2},"content":[{"type":"text","text":"502 Bad Gateway"}]},
  {"type":"calloutBlock","attrs":{"type":"warning"},"content":[{"type":"paragraph","content":[{"type":"text","text":"502 means the ingress controller received an invalid response from the upstream (backend) service."}]}]},
  {"type":"taskList","content":[
    {"type":"taskItem","attrs":{"checked":false},"content":[{"type":"paragraph","content":[{"type":"text","text":"Check if backend pods are running: kubectl get pods -n <namespace> -l app=<service>"}]}]},
    {"type":"taskItem","attrs":{"checked":false},"content":[{"type":"paragraph","content":[{"type":"text","text":"Check backend service endpoints: kubectl get endpoints <service> -n <namespace>"}]}]},
    {"type":"taskItem","attrs":{"checked":false},"content":[{"type":"paragraph","content":[{"type":"text","text":"Check ingress controller logs: kubectl logs -n ingress-nginx -l app.kubernetes.io/name=ingress-nginx"}]}]},
    {"type":"taskItem","attrs":{"checked":false},"content":[{"type":"paragraph","content":[{"type":"text","text":"Test direct connectivity to the backend pod: kubectl port-forward pod/<pod> 8080:8080"}]}]}
  ]},
  {"type":"heading","attrs":{"level":2},"content":[{"type":"text","text":"503 Service Unavailable"}]},
  {"type":"calloutBlock","attrs":{"type":"danger"},"content":[{"type":"paragraph","content":[{"type":"text","text":"503 means no backend endpoints are available to serve the request."}]}]},
  {"type":"bulletList","content":[
    {"type":"listItem","content":[{"type":"paragraph","content":[{"type":"text","text":"All backend pods may be down or failing readiness probes"}]}]},
    {"type":"listItem","content":[{"type":"paragraph","content":[{"type":"text","text":"The Service selector may not match any pods (check labels)"}]}]},
    {"type":"listItem","content":[{"type":"paragraph","content":[{"type":"text","text":"During a deployment rollout, old pods may be terminated before new ones are ready"}]}]}
  ]},
  {"type":"heading","attrs":{"level":2},"content":[{"type":"text","text":"504 Gateway Timeout"}]},
  {"type":"paragraph","content":[{"type":"text","text":"The backend did not respond within the configured timeout. Check application performance and increase timeout annotations if needed:"}]},
  {"type":"codeBlock","attrs":{"language":"yaml"},"content":[{"type":"text","text":"apiVersion: networking.k8s.io/v1\nkind: Ingress\nmetadata:\n  annotations:\n    nginx.ingress.kubernetes.io/proxy-read-timeout: \"120\"\n    nginx.ingress.kubernetes.io/proxy-send-timeout: \"120\"\n    nginx.ingress.kubernetes.io/proxy-connect-timeout: \"10\""}]},
  {"type":"heading","attrs":{"level":2},"content":[{"type":"text","text":"SSL/TLS Issues"}]},
  {"type":"taskList","content":[
    {"type":"taskItem","attrs":{"checked":false},"content":[{"type":"paragraph","content":[{"type":"text","text":"Check certificate status: kubectl get certificate -A"}]}]},
    {"type":"taskItem","attrs":{"checked":false},"content":[{"type":"paragraph","content":[{"type":"text","text":"Verify TLS secret exists and has correct data: kubectl get secret <tls-secret> -n <ns> -o yaml"}]}]},
    {"type":"taskItem","attrs":{"checked":false},"content":[{"type":"paragraph","content":[{"type":"text","text":"Check for certificate/key mismatch in ingress controller logs"}]}]},
    {"type":"taskItem","attrs":{"checked":false},"content":[{"type":"paragraph","content":[{"type":"text","text":"Test externally: curl -vI https://<domain> to check the presented certificate"}]}]}
  ]}
]}`

// --- Architecture & engineering docs ---

const adrKubernetesContent = `{"type":"doc","content":[
  {"type":"heading","attrs":{"level":1},"content":[{"type":"text","text":"ADR-001: Kubernetes as Container Orchestration Platform"}]},
  {"type":"heading","attrs":{"level":2},"content":[{"type":"text","text":"Status"}]},
  {"type":"paragraph","content":[{"type":"text","marks":[{"type":"bold"}],"text":"Accepted"},{"type":"text","text":" — 2025-06-15"}]},
  {"type":"heading","attrs":{"level":2},"content":[{"type":"text","text":"Context"}]},
  {"type":"paragraph","content":[{"type":"text","text":"We need a container orchestration platform that supports multi-tenant workloads, auto-scaling, rolling deployments, and integration with our existing monitoring stack (Prometheus, Grafana). Our team has experience with both Docker Swarm and Kubernetes."}]},
  {"type":"heading","attrs":{"level":2},"content":[{"type":"text","text":"Decision"}]},
  {"type":"paragraph","content":[{"type":"text","text":"We will use Kubernetes (managed via CNCF-certified distributions) for all production workloads. Specifically:"}]},
  {"type":"bulletList","content":[
    {"type":"listItem","content":[{"type":"paragraph","content":[{"type":"text","text":"Production: Managed Kubernetes (EKS/GKE) for reduced operational overhead"}]}]},
    {"type":"listItem","content":[{"type":"paragraph","content":[{"type":"text","text":"Development: k3s on local machines for fast iteration"}]}]},
    {"type":"listItem","content":[{"type":"paragraph","content":[{"type":"text","text":"GitOps: Argo CD for declarative cluster management"}]}]},
    {"type":"listItem","content":[{"type":"paragraph","content":[{"type":"text","text":"Packaging: Helm charts for all services"}]}]}
  ]},
  {"type":"heading","attrs":{"level":2},"content":[{"type":"text","text":"Consequences"}]},
  {"type":"bulletList","content":[
    {"type":"listItem","content":[{"type":"paragraph","content":[{"type":"text","text":"Positive: Industry-standard ecosystem with extensive tooling and community support"}]}]},
    {"type":"listItem","content":[{"type":"paragraph","content":[{"type":"text","text":"Positive: Native support for auto-scaling, rolling updates, and self-healing"}]}]},
    {"type":"listItem","content":[{"type":"paragraph","content":[{"type":"text","text":"Negative: Higher learning curve than simpler orchestrators"}]}]},
    {"type":"listItem","content":[{"type":"paragraph","content":[{"type":"text","text":"Negative: Requires dedicated platform engineering effort for cluster management"}]}]}
  ]}
]}`

const onboardingGuideContent = `{"type":"doc","content":[
  {"type":"heading","attrs":{"level":1},"content":[{"type":"text","text":"On-Call Engineer Onboarding Guide"}]},
  {"type":"paragraph","content":[{"type":"text","text":"Welcome to the on-call team! This guide covers everything you need to get started with on-call duties at Acme Corp."}]},
  {"type":"heading","attrs":{"level":2},"content":[{"type":"text","text":"Before Your First Shift"}]},
  {"type":"taskList","content":[
    {"type":"taskItem","attrs":{"checked":false},"content":[{"type":"paragraph","content":[{"type":"text","text":"Get access to NightOwl (alert management) and BookOwl (runbooks)"}]}]},
    {"type":"taskItem","attrs":{"checked":false},"content":[{"type":"paragraph","content":[{"type":"text","text":"Join the #ops-alerts Slack channel and configure mobile notifications"}]}]},
    {"type":"taskItem","attrs":{"checked":false},"content":[{"type":"paragraph","content":[{"type":"text","text":"Install kubectl and configure access to all production clusters"}]}]},
    {"type":"taskItem","attrs":{"checked":false},"content":[{"type":"paragraph","content":[{"type":"text","text":"Verify VPN access to the internal monitoring dashboards"}]}]},
    {"type":"taskItem","attrs":{"checked":false},"content":[{"type":"paragraph","content":[{"type":"text","text":"Bookmark Grafana dashboards: SLA Overview, Cluster Health, Service Latency"}]}]},
    {"type":"taskItem","attrs":{"checked":false},"content":[{"type":"paragraph","content":[{"type":"text","text":"Read through all runbooks in the On-Call Runbooks space"}]}]},
    {"type":"taskItem","attrs":{"checked":false},"content":[{"type":"paragraph","content":[{"type":"text","text":"Shadow an experienced engineer for at least one shift"}]}]}
  ]},
  {"type":"heading","attrs":{"level":2},"content":[{"type":"text","text":"During Your Shift"}]},
  {"type":"calloutBlock","attrs":{"type":"info"},"content":[{"type":"paragraph","content":[{"type":"text","text":"Your primary goal is to restore service, not to find the root cause. Escalate early rather than spending hours debugging alone."}]}]},
  {"type":"orderedList","content":[
    {"type":"listItem","content":[{"type":"paragraph","content":[{"type":"text","text":"When an alert fires, acknowledge it within 5 minutes in NightOwl"}]}]},
    {"type":"listItem","content":[{"type":"paragraph","content":[{"type":"text","text":"Check if a runbook exists for this alert type — follow it step by step"}]}]},
    {"type":"listItem","content":[{"type":"paragraph","content":[{"type":"text","text":"If no runbook exists, check the Knowledge Base for similar past incidents"}]}]},
    {"type":"listItem","content":[{"type":"paragraph","content":[{"type":"text","text":"Post updates in #ops-alerts every 15 minutes during active incidents"}]}]},
    {"type":"listItem","content":[{"type":"paragraph","content":[{"type":"text","text":"If unresolved after 15 minutes, escalate to the next tier (your manager will be notified automatically)"}]}]}
  ]},
  {"type":"heading","attrs":{"level":2},"content":[{"type":"text","text":"After Your Shift"}]},
  {"type":"bulletList","content":[
    {"type":"listItem","content":[{"type":"paragraph","content":[{"type":"text","text":"Hand off any open incidents to the next on-call engineer via the roster handoff in NightOwl"}]}]},
    {"type":"listItem","content":[{"type":"paragraph","content":[{"type":"text","text":"For any incident that lasted > 30 minutes, create a post-mortem document"}]}]},
    {"type":"listItem","content":[{"type":"paragraph","content":[{"type":"text","text":"If you encountered a scenario not covered by existing runbooks, write a new one"}]}]}
  ]},
  {"type":"heading","attrs":{"level":2},"content":[{"type":"text","text":"Escalation Path"}]},
  {"type":"paragraph","content":[{"type":"text","text":"The escalation policy is configured in NightOwl and follows three tiers:"}]},
  {"type":"bulletList","content":[
    {"type":"listItem","content":[{"type":"paragraph","content":[{"type":"text","marks":[{"type":"bold"}],"text":"Tier 1 (0-5 min):"},{"type":"text","text":" On-call engineer — Slack + SMS notification"}]}]},
    {"type":"listItem","content":[{"type":"paragraph","content":[{"type":"text","marks":[{"type":"bold"}],"text":"Tier 2 (5-15 min):"},{"type":"text","text":" On-call + engineering manager — Slack + SMS + phone call"}]}]},
    {"type":"listItem","content":[{"type":"paragraph","content":[{"type":"text","marks":[{"type":"bold"}],"text":"Tier 3 (15-30 min):"},{"type":"text","text":" VP Engineering — phone call"}]}]}
  ]}
]}`

const deploymentSOPContent = `{"type":"doc","content":[
  {"type":"heading","attrs":{"level":1},"content":[{"type":"text","text":"Production Deployment Procedure"}]},
  {"type":"calloutBlock","attrs":{"type":"info"},"content":[{"type":"paragraph","content":[{"type":"text","text":"Standard operating procedure for deploying services to production. All deployments must follow this checklist."}]}]},
  {"type":"heading","attrs":{"level":2},"content":[{"type":"text","text":"Pre-Deployment Checklist"}]},
  {"type":"taskList","content":[
    {"type":"taskItem","attrs":{"checked":false},"content":[{"type":"paragraph","content":[{"type":"text","text":"All CI checks pass (unit tests, integration tests, lint, security scan)"}]}]},
    {"type":"taskItem","attrs":{"checked":false},"content":[{"type":"paragraph","content":[{"type":"text","text":"PR approved by at least 2 reviewers"}]}]},
    {"type":"taskItem","attrs":{"checked":false},"content":[{"type":"paragraph","content":[{"type":"text","text":"Database migrations tested in staging (if any)"}]}]},
    {"type":"taskItem","attrs":{"checked":false},"content":[{"type":"paragraph","content":[{"type":"text","text":"Feature flags configured for gradual rollout (if applicable)"}]}]},
    {"type":"taskItem","attrs":{"checked":false},"content":[{"type":"paragraph","content":[{"type":"text","text":"Rollback plan documented and tested"}]}]},
    {"type":"taskItem","attrs":{"checked":false},"content":[{"type":"paragraph","content":[{"type":"text","text":"No active incidents on affected services"}]}]},
    {"type":"taskItem","attrs":{"checked":false},"content":[{"type":"paragraph","content":[{"type":"text","text":"Deploy within change window (Tuesday-Thursday, 09:00-15:00 UTC)"}]}]}
  ]},
  {"type":"heading","attrs":{"level":2},"content":[{"type":"text","text":"Deployment Steps"}]},
  {"type":"orderedList","content":[
    {"type":"listItem","content":[{"type":"paragraph","content":[{"type":"text","text":"Announce deployment in #ops-deploys: service name, version, and expected impact"}]}]},
    {"type":"listItem","content":[{"type":"paragraph","content":[{"type":"text","text":"Tag the release in Git and push the tag to trigger the CI/CD pipeline"}]}]},
    {"type":"listItem","content":[{"type":"paragraph","content":[{"type":"text","text":"Argo CD will detect the new image tag and begin rolling update"}]}]},
    {"type":"listItem","content":[{"type":"paragraph","content":[{"type":"text","text":"Monitor the rollout: kubectl rollout status deployment/<name> -n <namespace>"}]}]},
    {"type":"listItem","content":[{"type":"paragraph","content":[{"type":"text","text":"Watch error rates and latency in Grafana for 10 minutes after rollout completes"}]}]},
    {"type":"listItem","content":[{"type":"paragraph","content":[{"type":"text","text":"If metrics are healthy, announce deployment complete in #ops-deploys"}]}]}
  ]},
  {"type":"heading","attrs":{"level":2},"content":[{"type":"text","text":"Rollback Procedure"}]},
  {"type":"calloutBlock","attrs":{"type":"danger"},"content":[{"type":"paragraph","content":[{"type":"text","text":"If error rates increase by more than 2x or latency exceeds SLO, roll back immediately."}]}]},
  {"type":"codeBlock","attrs":{"language":"bash"},"content":[{"type":"text","text":"# Option 1: Argo CD rollback\nargocd app rollback <app-name>\n\n# Option 2: kubectl rollback\nkubectl rollout undo deployment/<name> -n <namespace>\n\n# Verify rollback\nkubectl rollout status deployment/<name> -n <namespace>"}]},
  {"type":"heading","attrs":{"level":2},"content":[{"type":"text","text":"Post-Deployment"}]},
  {"type":"bulletList","content":[
    {"type":"listItem","content":[{"type":"paragraph","content":[{"type":"text","text":"Update the deployment log spreadsheet with version, deployer, and timestamp"}]}]},
    {"type":"listItem","content":[{"type":"paragraph","content":[{"type":"text","text":"If a rollback occurred, create an incident and schedule a post-mortem"}]}]},
    {"type":"listItem","content":[{"type":"paragraph","content":[{"type":"text","text":"Close the deployment thread in #ops-deploys"}]}]}
  ]}
]}`

const secretsRotationContent = `{"type":"doc","content":[
  {"type":"heading","attrs":{"level":1},"content":[{"type":"text","text":"Secrets and Credential Rotation"}]},
  {"type":"paragraph","content":[{"type":"text","text":"This runbook covers the process for rotating secrets, API keys, database passwords, and TLS certificates across our infrastructure."}]},
  {"type":"heading","attrs":{"level":2},"content":[{"type":"text","text":"Database Passwords"}]},
  {"type":"calloutBlock","attrs":{"type":"warning"},"content":[{"type":"paragraph","content":[{"type":"text","text":"Database password rotation requires a maintenance window. Both the old and new passwords must work simultaneously during the transition."}]}]},
  {"type":"orderedList","content":[
    {"type":"listItem","content":[{"type":"paragraph","content":[{"type":"text","text":"Create the new password in the secret manager (AWS Secrets Manager / Vault)"}]}]},
    {"type":"listItem","content":[{"type":"paragraph","content":[{"type":"text","text":"Add the new password as an additional valid credential in PostgreSQL: ALTER ROLE <role> PASSWORD '<new>'"}]}]},
    {"type":"listItem","content":[{"type":"paragraph","content":[{"type":"text","text":"Update the Kubernetes secret with the new password"}]}]},
    {"type":"listItem","content":[{"type":"paragraph","content":[{"type":"text","text":"Rolling restart all pods that use this credential"}]}]},
    {"type":"listItem","content":[{"type":"paragraph","content":[{"type":"text","text":"Verify all pods are connecting successfully with the new password"}]}]},
    {"type":"listItem","content":[{"type":"paragraph","content":[{"type":"text","text":"Remove the old password from PostgreSQL"}]}]}
  ]},
  {"type":"heading","attrs":{"level":2},"content":[{"type":"text","text":"API Keys (NightOwl / BookOwl)"}]},
  {"type":"orderedList","content":[
    {"type":"listItem","content":[{"type":"paragraph","content":[{"type":"text","text":"Generate a new API key in the admin panel of the service"}]}]},
    {"type":"listItem","content":[{"type":"paragraph","content":[{"type":"text","text":"Update the consuming service's environment variables or Kubernetes secret"}]}]},
    {"type":"listItem","content":[{"type":"paragraph","content":[{"type":"text","text":"Restart the consuming service pods"}]}]},
    {"type":"listItem","content":[{"type":"paragraph","content":[{"type":"text","text":"Verify integration is working (check integration status page)"}]}]},
    {"type":"listItem","content":[{"type":"paragraph","content":[{"type":"text","text":"Revoke the old API key from the admin panel"}]}]}
  ]},
  {"type":"heading","attrs":{"level":2},"content":[{"type":"text","text":"Image Pull Secrets (GHCR)"}]},
  {"type":"codeBlock","attrs":{"language":"bash"},"content":[{"type":"text","text":"# Generate new token in GitHub > Settings > Developer Settings > PAT\n# Then update the Kubernetes secret:\nkubectl create secret docker-registry ghcr-pull \\\n  --docker-server=ghcr.io \\\n  --docker-username=<github-user> \\\n  --docker-password=<new-pat> \\\n  -n <namespace> \\\n  --dry-run=client -o yaml | kubectl apply -f -\n\n# Verify: try pulling an image manually\nkubectl run pull-test --image=ghcr.io/acme/test:latest --restart=Never\nkubectl delete pod pull-test"}]},
  {"type":"heading","attrs":{"level":2},"content":[{"type":"text","text":"Rotation Schedule"}]},
  {"type":"bulletList","content":[
    {"type":"listItem","content":[{"type":"paragraph","content":[{"type":"text","marks":[{"type":"bold"}],"text":"Database passwords:"},{"type":"text","text":" Every 90 days"}]}]},
    {"type":"listItem","content":[{"type":"paragraph","content":[{"type":"text","marks":[{"type":"bold"}],"text":"API keys:"},{"type":"text","text":" Every 180 days or after personnel changes"}]}]},
    {"type":"listItem","content":[{"type":"paragraph","content":[{"type":"text","marks":[{"type":"bold"}],"text":"Image pull tokens:"},{"type":"text","text":" Every 90 days (set GitHub PAT expiry to match)"}]}]},
    {"type":"listItem","content":[{"type":"paragraph","content":[{"type":"text","marks":[{"type":"bold"}],"text":"TLS certificates:"},{"type":"text","text":" Automated via cert-manager (90-day Let's Encrypt certs, renewed at 60 days)"}]}]}
  ]}
]}`

const monitoringGuideContent = `{"type":"doc","content":[
  {"type":"heading","attrs":{"level":1},"content":[{"type":"text","text":"Monitoring and Alerting Standards"}]},
  {"type":"paragraph","content":[{"type":"text","text":"This document defines our monitoring standards, alert thresholds, and dashboard conventions. All services must implement these standards before going to production."}]},
  {"type":"heading","attrs":{"level":2},"content":[{"type":"text","text":"Required Metrics (RED Method)"}]},
  {"type":"paragraph","content":[{"type":"text","text":"Every service must expose these three metric types via a /metrics Prometheus endpoint:"}]},
  {"type":"bulletList","content":[
    {"type":"listItem","content":[{"type":"paragraph","content":[{"type":"text","marks":[{"type":"bold"}],"text":"Rate:"},{"type":"text","text":" Request throughput (requests per second)"}]}]},
    {"type":"listItem","content":[{"type":"paragraph","content":[{"type":"text","marks":[{"type":"bold"}],"text":"Errors:"},{"type":"text","text":" Error rate as a percentage of total requests (5xx responses)"}]}]},
    {"type":"listItem","content":[{"type":"paragraph","content":[{"type":"text","marks":[{"type":"bold"}],"text":"Duration:"},{"type":"text","text":" Request latency histograms (p50, p95, p99)"}]}]}
  ]},
  {"type":"heading","attrs":{"level":2},"content":[{"type":"text","text":"Alert Severity Levels"}]},
  {"type":"bulletList","content":[
    {"type":"listItem","content":[{"type":"paragraph","content":[{"type":"text","marks":[{"type":"bold"}],"text":"Critical:"},{"type":"text","text":" Customer-facing impact, pages on-call immediately. Examples: service down, data loss risk, security breach."}]}]},
    {"type":"listItem","content":[{"type":"paragraph","content":[{"type":"text","marks":[{"type":"bold"}],"text":"Warning:"},{"type":"text","text":" Degraded performance or approaching limits. Will become critical if not addressed. Examples: disk > 80%, high latency, certificate expiring."}]}]},
    {"type":"listItem","content":[{"type":"paragraph","content":[{"type":"text","marks":[{"type":"bold"}],"text":"Info:"},{"type":"text","text":" Informational, no action required. Sent to Slack only. Examples: deployment completed, autoscaling event."}]}]}
  ]},
  {"type":"heading","attrs":{"level":2},"content":[{"type":"text","text":"Standard Alert Rules"}]},
  {"type":"codeBlock","attrs":{"language":"yaml"},"content":[{"type":"text","text":"# Example: PrometheusRule for a Go HTTP service\napiVersion: monitoring.coreos.com/v1\nkind: PrometheusRule\nmetadata:\n  name: my-service-alerts\nspec:\n  groups:\n  - name: my-service\n    rules:\n    - alert: HighErrorRate\n      expr: |\n        sum(rate(http_requests_total{status=~\"5..\",service=\"my-service\"}[5m]))\n        / sum(rate(http_requests_total{service=\"my-service\"}[5m])) > 0.02\n      for: 5m\n      labels:\n        severity: critical\n      annotations:\n        summary: \"Error rate > 2% for my-service\"\n    - alert: HighLatency\n      expr: |\n        histogram_quantile(0.99, rate(http_request_duration_seconds_bucket{service=\"my-service\"}[5m])) > 2\n      for: 10m\n      labels:\n        severity: warning\n      annotations:\n        summary: \"p99 latency > 2s for my-service\""}]},
  {"type":"heading","attrs":{"level":2},"content":[{"type":"text","text":"Dashboard Conventions"}]},
  {"type":"orderedList","content":[
    {"type":"listItem","content":[{"type":"paragraph","content":[{"type":"text","text":"Every service dashboard starts with the RED metrics (rate, errors, duration) in the top row"}]}]},
    {"type":"listItem","content":[{"type":"paragraph","content":[{"type":"text","text":"Use consistent time ranges: default 6h, with quick selectors for 1h, 24h, 7d"}]}]},
    {"type":"listItem","content":[{"type":"paragraph","content":[{"type":"text","text":"Include a resource usage row: CPU, memory, network I/O, disk (if stateful)"}]}]},
    {"type":"listItem","content":[{"type":"paragraph","content":[{"type":"text","text":"Tag all dashboards with the service name and team for discoverability"}]}]}
  ]}
]}`
