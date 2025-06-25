resource "castai_workload_scaling_policy_order" "custom" {
  cluster_id = castai_gke_cluster.dev.id
  policy_ids = [
    "be61e44b-0f7c-44da-b60e-0594d6fd3634",
    "f60b79f0-d21e-4eda-a59f-f5daa846d289",
    "834e4f2c-0d67-4f97-9898-77ec9350fda8",
    "d2a79ea2-91eb-4737-b719-c60b9abffe36",
    "c51810d9-f020-47e0-836f-047789d2e900",
    "23f8c959-28fa-454e-b193-5255c4334946",
    "b239ebdb-1cd0-454d-a3d1-faf6842d48b6",
    "7d713728-9ffd-4b9e-9ca9-7a8a19f2b701",
    "4f1f625f-4b63-4047-89c9-b945d28a701a",
    "9040688a-ae73-474c-bad5-aadd72a14ac4",
    "7162cde3-8f17-459e-b308-a4eb2a264364",
  ]
}
