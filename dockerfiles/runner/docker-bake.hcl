
variable "BUILD_VERSION" {
    default = "dev"
}

group "default" {
    targets = ["runner-16", "runner-15", "runner-14", "runner-13"]
}

target "runner-16" {
    inherits = ["docker-metadata-action", "base-runner"]

    contexts = {
        postgres_base = "docker-image://postgres:16-bookworm"
    }
}

target "runner-15" {
    inherits = ["docker-metadata-action", "base-runner"]

    contexts = {
        postgres_base = "docker-image://postgres:15-bookworm"
    }
}

target "runner-14" {
    inherits = ["docker-metadata-action", "base-runner"]

    contexts = {
        postgres_base = "docker-image://postgres:14-bookworm"
    }
}

target "runner-13" {
    inherits = ["docker-metadata-action", "base-runner"]

    contexts = {
        postgres_base = "docker-image://postgres:13-bookworm"
    }
}

target "pgxman" {
    dockerfile = "dockerfiles/shared/Dockerfile.pgxman"
    target = "gobuild"

    args = {
        BUILD_VERSION = "${BUILD_VERSION}"
    }
}

target "base-runner" {
    contexts = {
        pgxman = "target:pgxman"
    }

    dockerfile = "dockerfiles/runner/Dockerfile.postgres"
}

# Inherit this target for CI use
# Ref https://github.com/docker/metadata-action#bake-definition
target "docker-metadata-action" {}
