# Associate terraform resource "my_schedule" with a hibernation schedule named "workday".
# Will use the default organization ID, that is associated with the API token.
terraform import 'castai_hibernation_schedule.my_schedule' workday

# Importing via direct schedule ID is also possible.
# Will use the default organization ID, that is associated with the API token.
terraform import 'castai_hibernation_schedule.my_schedule' e5ee784d-2c4b-4820-ab4e-16e4b81534a4

# Import using organization ID and schedule name/ID in format <organization id>/<schedule id|schedule name>.
terraform import 'castai_hibernation_schedule.my_schedule' 63895dfe-dc8b-49b6-959c-3f3545de525a/workday
