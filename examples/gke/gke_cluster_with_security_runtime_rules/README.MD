# Example of GKE cluster connected to CAST AI with enabled Kvisor security agent and customized runtime security rules
Following example creates GKE cluster and its supporting resources using GKE community modules.
After GKE cluster is created it is onboarded to CAST AI.
[Kvisor security agent](https://docs.cast.ai/docs/kvisor) is deployed to the cluster and security policies are enabled.\
See `install_security_agent` and `kvisor_values` variables in `castai.tf` file enabling Kvisor.\
See`castai_runtime_rule` resource defining each rule based on castai_runtime_rule.yaml file.\
Example configuration should be analysed in the following order:
1. Create VPC - `vpc.tf`
2. Create GKE cluster - `gke.tf`
3. Create CAST AI related resources to connect GKE cluster to CAST AI with Kvisor and runtime rules - `castai.tf`\
Note that Kvisor runtime security rules are stored in a file: `castai_runtime_rules.yaml`
4. If needed import existing CAST AI runtime rules with `fetch_castai_runtime_rules.sh` script.\
   In order to import - set your API key and optionally the base URL: 
   ```
   export CASTAI_API_KEY="your-api-key-here"
   export CASTAI_BASE_URL="https://api.cast.ai"  # Optional if default is fine
   ```
   Run the script (it requires yq and jq installed):
   ```
   # Fetch all rules
   ./fetch_castai_runtime_rules.sh
   
   # Fetch only already enabled rules
   ./fetch_castai_runtime_rules.sh --enabled-only
   ```
   Output:\
   YAML saved to `castai_runtime_rule_new.yaml`\
   Copy wanted runtime rules from `castai_runtime_rule_new.yaml` to `castai_runtime_rule.yaml` file.\
   Terraform import commands are printed to console, you need to import newly added resources to terraform state before running any terraform commands.

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
