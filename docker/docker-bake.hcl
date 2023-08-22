
variable "BUILD_VERSION" {
    default = "dev"
}

group "default" {
    targets = ["debian-bookworm", "ubuntu-jammy"]
}

target "debian-bookworm" {
    inherits = ["docker-metadata-action"]

    contexts = {
        pgxman = "target:pgxman"
        debian_base = "docker-image://postgres:15-bookworm"
    }

    dockerfile = "docker/Dockerfile.debian"
}

target "ubuntu-jammy" {
    inherits = ["docker-metadata-action"]

    contexts = {
        pgxman = "target:pgxman"
        debian_base = "docker-image://ubuntu:jammy"
    }

    dockerfile = "docker/Dockerfile.debian"
}

target "pgxman" {
    dockerfile = "docker/Dockerfile.pgxman"
    target = "gobuild"

    args = {
        BUILD_VERSION = "${BUILD_VERSION}"
    }
}

# Inherit this target for CI use
# Ref https://github.com/docker/metadata-action#bake-definition
target "docker-metadata-action" {}
