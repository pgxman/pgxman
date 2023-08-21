
variable "BASE_REPO" {
    default = "ghcr.io/pgxman/builder"
}

variable "DEBIAN_BOOKWORM_REPO" {
    default = "${BASE_REPO}/debian/bookworm"
}

variable "UBUNTU_JAMMY_REPO" {
    default = "${BASE_REPO}/ubuntu/jammy"
}

variable "TAG" {
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
    tags = ["${DEBIAN_BOOKWORM_REPO}:${TAG}"]
}

target "ubuntu-jammy" {
    inherits = ["docker-metadata-action"]

    contexts = {
        pgxman = "target:pgxman"
        debian_base = "docker-image://ubuntu:jammy"
    }

    dockerfile = "docker/Dockerfile.debian"
    tags = ["${UBUNTU_JAMMY_REPO}:${TAG}"]
}

target "pgxman" {
    dockerfile = "docker/Dockerfile.pgxman"
    target = "gobuild"

    args = {
        BUILD_VERSION = "${TAG}"
    }
}

# Inherit this target for CI use
# Ref https://github.com/docker/metadata-action#bake-definition
target "docker-metadata-action" {}
