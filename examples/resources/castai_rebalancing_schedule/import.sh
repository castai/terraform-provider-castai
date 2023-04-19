
# associate terraform resource "spots" with a rebalancing schedule named "spots"
terraform import 'castai_rebalancing_schedule.spots' spots

# importing via direct schedule ID is also possible
terraform import 'castai_rebalancing_schedule.spots' b4e69e0c-1762-45eb-bd4f-85cb172e6ad3
