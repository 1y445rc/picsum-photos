terraform {
  required_version = ">= 1.3.0"

  required_providers {
    digitalocean = {
      source = "digitalocean/digitalocean"
      version = "2.23.0"
    }

    kubernetes = {
      source = "hashicorp/kubernetes"
      version = "2.14.0"
    }
  }
}
