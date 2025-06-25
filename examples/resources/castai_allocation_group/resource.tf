resource "castai_allocation_group" "ag_example" {
  name = "ag_example"
  cluster_ids = [
    "1a58d6b4-bc0e-4417-b9c7-31d15c313f3f",
    "d204b988-5db5-472e-a258-bf763a0f4a93"
  ]

  namespaces = [
    "namespace-a",
    "namespace-b"
  ]

  labels = {
    environment              = "production",
    team                     = "my-team",
    "app.kubernetes.io/name" = "app-name"
  }

  labels_operator = "AND"
}
