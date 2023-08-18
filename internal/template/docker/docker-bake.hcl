variable "REPO" {
    default = "ghcr.io/pgxman/builder"
}

variable "TAG" {
    default = "main"
}

target "export" {
    contexts = {
        debian-bookworm = "target:debian-bookworm"
        ubuntu-jammy = "target:ubuntu-jammy"
    }

    dockerfile = "Dockerfile.export"
    target = "export"
}

target "debian-bookworm" {
    args = {
        BUILD_IMAGE = "${REPO}/debian/bookworm:${TAG}"
    }

    dockerfile = "Dockerfile"
    target = "build"
}

target "ubuntu-jammy" {
    args = {
        BUILD_IMAGE = "${REPO}/ubuntu/jammy:${TAG}"
    }

    dockerfile = "Dockerfile"
    target = "build"
}
