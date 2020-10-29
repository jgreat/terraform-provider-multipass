terraform {
  required_providers {
    multipass = {
      versions = ["0.0.1"]
      source   = "github.com/jgreat/multipass"
    }
  }
}

provider "multipass" {}

data "multipass_instance" "reelhealth" {}

output "instance" {
  value = data.multipass_instance.reelhealth
}
