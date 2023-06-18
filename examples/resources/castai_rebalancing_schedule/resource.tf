resource "castai_rebalancing_schedule" "spots" {
	name = "rebalance spots at every 30th minute"
	schedule {
		cron = "*/30 * * * *"
	}
	trigger_conditions {
		savings_percentage = 20
	}
	launch_configuration {
		# only consider instances older than 5 minutes
		node_ttl_seconds = 300
		num_targeted_nodes = 3
		rebalancing_min_nodes = 2
		selector = jsonencode({
			nodeSelectorTerms = [{
				matchExpressions = [
					{
						key =  "scheduling.cast.ai/spot"
						operator = "Exists"
					}
				]
			}]
		})
		execution_conditions {
			enabled = true
			achieved_savings_percentage = 10
		}
	}
}
