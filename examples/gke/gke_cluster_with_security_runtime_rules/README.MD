# Example of GKE cluster connected to CAST AI with enabled Kvisor security agent and customized runtime security rules
Following example creates GKE cluster and its supporting resources using GKE community modules.
After GKE cluster is created it is onboarded to CAST AI.
[Kvisor security agent](https://docs.cast.ai/docs/kvisor) is deployed to the cluster and security policies are enabled.\
See `install_security_agent` and `kvisor_values` variables in `castai.tf` file enabling Kvisor.\
See `castai_security_runtime_rule` resource defining example [Runtime security rule](https://docs.cast.ai/docs/anomaly-rules-engine) in `castai_runtime_rules.tf` file.\
Example configuration should be analysed in the following order:
1. Create VPC - `vpc.tf`
2. Create GKE cluster - `gke.tf`
3. Create CAST AI related resources to connect GKE cluster to CAST AI with [Kvisor security agent](https://docs.cast.ai/docs/kvisor) enabled - `castai.tf`
4. Create CAST AI Runtime security rules - `castai_runtime_rules.tf`
5. If needed - import existing CAST AI runtime rules with `fetch_castai_runtime_rules.sh` script.\
   Script will generate terraform resources and will print terraform import commands to the console.\
\
   In order to import - set your API key and optionally the base URL: 
   ```
   export CASTAI_API_KEY="your-api-key-here"
   export CASTAI_BASE_URL="https://api.cast.ai"  # Optional, default is https://api.cast.ai
   ```
   Run the script (it requires jq installed):
   ```
   # Fetch all rules
   ./fetch_castai_runtime_rules.sh
   
   # Fetch only already enabled rules
   ./fetch_castai_runtime_rules.sh --enabled-only
   ```
   Output:\
   Terraform resources are saved to `castai_runtime_rules.tf.example`\
   Copy wanted runtime rules from `castai_runtime_rules.tf.example` to `castai_runtime_rules.tf` file.\
   Terraform import commands are printed to console, you need to import newly added resources to terraform state before running any other commands.

# Terraform Usage
1. Rename `tf.vars.example` to `tf.vars`
2. Update `tf.vars` file with your project name, cluster name, cluster region and CAST AI API token.
3. Initialize Terraform. Under example root folder run:
```
terraform init
```
4. Run Terraform apply:
```
terraform apply -var-file=tf.vars
```
5. To destroy resources created by this example:
```
terraform destroy -var-file=tf.vars
```

Please refer to this guide if you run into any issues https://docs.cast.ai/docs/terraform-troubleshooting
