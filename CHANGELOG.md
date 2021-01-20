## 0.3.0 (January 20, 2021)

NOTES:

* DigitalOcean cloud is now supported
* CAST.AI api spec is now downloaded and regenerated on `make build`.

IMPROVEMENTS:

* `castai_credentials`: now allows `do` block for DigitalOcean credentials. Usage:

```
resource "castai_credentials" "example_do" {
  name = "example-do"
  do {
    token = var.do_token
  }
}
```

## 0.2.2 (January 6, 2021)

NOTES:

* switched to https://api.cast.ai/v1 url.

## 0.1.0 (October 19, 2020)

FEATURES:

* **New Datasource:** `castai_credentials`
* **New Datasource:** `castai_cluster`
* **New Resource:** `castai_credentials`
* **New Resource:** `castai_cluster`

<!--- 
release notes format to follow: https://github.com/digitalocean/terraform-provider-digitalocean/blob/master/CHANGELOG.md
-->
