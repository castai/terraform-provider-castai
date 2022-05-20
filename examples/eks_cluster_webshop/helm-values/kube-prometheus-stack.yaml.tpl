grafana:
  adminPassword: "${ grafana_password }"
  dashboardProviders:
    dashboardproviders.yaml:
      apiVersion: 1
      providers:
      - name: 'default'
        orgId: 1
        folder: ''
        type: file
        disableDeletion: false
        editable: true
        options:
          path: /var/lib/grafana/dashboards/default

  dashboards:
    default:
        nginx-ingress:
            gnetId: 9614
            datasource: Prometheus
  additionalDataSources:
   - name: loki
     access: proxy
     basicAuth: false
     editable: false
     jsonData:
         tlsSkipVerify: true
     type: loki
     url: http://loki-read-headless:3100
  ingress:
      ingressClassName: nginx
      # Use cert-manager for HTTPS
      # annotations:
      #  cert-manager.io/cluster-issuer: selfsigned
      enabled: false
prometheus:
  prometheusSpec:
    serviceMonitorSelectorNilUsesHelmValues: false
    podMonitorSelectorNilUsesHelmValues: false
    probeSelectorNilUsesHelmValues: false
    storageSpec:
      volumeClaimTemplate:
        spec:
          storageClassName: ebs-sc
          accessModes: [ "ReadWriteOnce" ]
          resources:
            requests:
              storage: 10Gi


