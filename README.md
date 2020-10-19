<a href="https://cast.ai">
    <img src="https://cast.ai/wp-content/themes/cast/assets/img/cast-logo-dark-blue.svg" align="right" height="100" />
</a>

Terraform Provider for CAST.AI
==================


Website: https://www.cast.ai

[![Build Status](https://github.com/castai/terraform-provider-castai/workflows/Build/badge.svg)](https://github.com/castai/terraform-provider-castai/actions)



Requirements
------------

- [Terraform](https://www.terraform.io/downloads.html) 0.13+
- [Go](https://golang.org/doc/install) 1.15 (to build the provider plugin)

Using the provider
----------------------

To install this provider, put the following code into your Terraform configuration. Then, run `terraform init`.

```
terraform {
  required_providers {
    castai = {
      source  = "castai/castai"
      version = "0.1.0" # can be omitted for the latest version
    }
  }
  required_version = ">= 0.13"
}

provider "castai" {

}
```

To configure the provider with your personal api key, define the `api_key` in the provider.
```
provider "castai" {
  api_key = "<<your-castai-api-key>>"
}
```
Alternatively, you can pass api key via environment variable:
```sh
$ CASTAI_API_TOKEN=<<your-castai-api-key>> terraform plan
```

For more logs use the log level flag: 

```sh
$ TF_LOG=DEBUG terraform plan
```



More examples can be found [here](examples/).

_Learn why `required_providers` block is required in [terraform 0.13 upgrade guide](https://www.terraform.io/upgrade-guides/0-13.html#explicit-provider-source-locations)._

Developing the provider
---------------------------

Make sure you have [Go](http://www.golang.org) installed on your machine (please check the [requirements](#requirements)).

To build the provider locally:
```sh
$ git clone https://github.com/CastAI/terraform-provider-castai.git
$ cd terraform-provider-castai
$ make build
```

_`make build` builds the provider and install symlinks to that build for all terraform projects in `examples/*` dir.
Now you can work on `examples/localdev`._

Whenever you make changes to the provider re-run `make build`. 

You'll need to run `terraform init` in your terraform project again since the binary has changed.

To run unit tests:
```sh
$ make test
```

Releasing the Provider
----------------------

The provider is still in early 0.x.x stage. At the moment, releases will be performed manually for tags matching `vX.Y.Z`.

Once releases appear in GitHub Releases page, they will become available via the Terraform Registry.




