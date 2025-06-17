# AWS Live Migration with Containerd

This setup creates an EKS cluster and onboards it to the CAST AI. Live binaries are then installed on nodes using dedicated
Node Configuration. Included script installs Live binaries on nodes works with Amazon Linux 2023.

## How to create your env
1. Rename `tf.vars.example` to `tf.vars`
2. Update `tf.vars` file with your project name, cluster name, cluster region and Cast AI API token.
3. Initialize tofu. Under example root folder run:
    ```bash
    tofu init
    ```
4. Verify:
    ```
    tofu plan -var-file=tf.vars
    ```

5. Run tofu apply:
    ```
    tofu apply -var-file=tf.vars
    ```
6. To destroy resources created by this example:
    ```
    tofu destroy -var-file=tf.vars
    ```
   
## Troubleshooting
There are some known issues with the terraform setup, and know workarounds. 

### Cluster creation stuck / timeouts on node group creation
If cluster creation gets stuck on node group creation, and nodes are not healthy, it most probably means Calico installtion did not trigger
at the right time. To fix it, just break the tofu execution and reexecute it again.

### CAST AI onboarding stuck in connecting / pods don't have internet connection
Make sure Calico pods are running on all the nodes without errors and Core DNS addon is installed.

### Timeout on resources destruction
- Check if There are no hanging CAST AI EC2 instances left and blocking VPC deletion.
- If Calico uninstallation job is stuck for any reason, just delete it manually:
  ```bash
   k delete job -n tigera-operator tigera-operator-uninstall
  ```
### No AWS or tofu binaries

#### Setup AWS CLI
 - Follow the [installation guide](https://castai.atlassian.net/wiki/spaces/ENG/pages/2784493777/AWS) to install AWS CLI.
 
#### Setup tofu
 - For tofu run `brew install opentofu`
 - export AWS profile so tofu can pick it up: `export AWS_PROFILE=<ProfileName>`

## Enjoy
Once cluster is created and onboarded, you can manually play with Live Migrations.
