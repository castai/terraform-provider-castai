write:
  persistence:
    size: 10Gi
    storageClass: ebs-sc
  resources:
    limits:
      cpu: "1"
      memory: 1Gi
    requests:
      cpu: "0.5"
      memory: 1Gi


read:
  persistence:
    size: 10Gi
    storageClass: ebs-sc
  resources:
    limits:
      cpu: "1"
      memory: 1Gi
    requests:
      cpu: "0.5"
      memory: 1Gi

serviceMonitor:
  enabled: true

serviceAccount:
  annotations:
    "eks.amazonaws.com/role-arn": "${ loki_role_arn }"

loki:
  auth_enabled: false
  storage:
    bucketNames:
      chunks: "${ bucket_name }"
      ruler: ${ bucket_name }
    type: s3
    s3:
      s3: "${ s3_path }"
      endpoint: null
      accessKeyId: null
      secretAccessKey: null
      region: null
  storageConfig:
    boltdb_shipper:
     shared_store: s3
     active_index_directory: /var/loki/boltdb-shipper-active
     cache_location: /var/loki/boltdb-shipper-cache
     cache_ttl: 24h

  schemaConfig:
    configs:
      - from: 2020-05-15
        store: boltdb-shipper
        object_store: s3
        schema: v11
        index:
          prefix: loki_
          period: 24h

