terraform {
  required_providers {
    castai = {
      source = "castai/castai"
      # No version pin: under the kimchi DAP debugger the provider is launched via
      # --debug + TF_REATTACH_PROVIDERS; outside the debugger terraform dev_overrides
      # (see ~/.terraformrc) loads the locally built binary at the repo root.
    }
  }

  required_version = ">= 1.3.2"
}
