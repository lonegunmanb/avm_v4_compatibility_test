data "terraform" azurerm {
}

locals {
  updated_required_providers = try(join("\n", [for key, provider in data.terraform.azurerm.required_providers : "${key} = {\n  source = \"${provider.source}\"\n  version = \"${provider.source == "hashicorp/azurerm" ? "4.0.1" : provider.version}\"\n}"]), null)
}

transform "remove_block_element" terraform_required_providers {
  for_each             = local.updated_required_providers == null ? [] : ["remove_terraform_required_providers"]
  target_block_address = "terraform"
  paths                = ["required_providers"]
}

transform "concat_block_body" terraform_required_providers {
  for_each             = local.updated_required_providers == null ? [] : ["concat_terraform_required_providers"]
  target_block_address = "terraform"
  block_body           = "required_providers {\n${local.updated_required_providers}\n}"
  depends_on           = [transform.remove_block_element.terraform_required_providers]
}