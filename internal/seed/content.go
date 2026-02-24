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
