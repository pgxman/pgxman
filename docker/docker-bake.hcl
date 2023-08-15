
variable "REPO" {
    default = "ghcr.io/pgxman/builder"
}

variable "BUILD_VERSION" {
    default = "dev"
}

group "default" {
    targets = ["debian-bookworm", "ubuntu-jammy"]
}

target "debian-bookworm" {
    contexts = {
        pgxman = "target:pgxman"
        debian_base = "docker-image://postgres:15-bookworm"
    }

    dockerfile = "docker/Dockerfile.debian"
    tags = ["${REPO}/debian/bookworm"]
}

target "ubuntu-jammy" {
    contexts = {
        pgxman = "target:pgxman"
        debian_base = "docker-image://ubuntu:jammy"
    }

    dockerfile = "docker/Dockerfile.debian"
    tags = ["${REPO}/ubuntu/jammy"]
}

target "pgxman" {
    dockerfile = "docker/Dockerfile.pgxman"
    target = "gobuild"

    args = {
        BUILD_VERSION = "${BUILD_VERSION}"
    }
}
