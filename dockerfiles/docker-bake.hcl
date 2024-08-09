
variable "BUILD_VERSION" {
    default = "dev"
}

group "default" {
    targets = ["builder", "runner"]
}

group "builder" {
    targets = ["builder-debian-bookworm", "builder-ubuntu-jammy", "builder-ubuntu-noble"]
}

group "runner" {
    targets = ["runner-postgres-16", "runner-postgres-15", "runner-postgres-14", "runner-postgres-13"]
}

target "builder-debian-bookworm" {
    inherits = ["docker-metadata-action", "base-builder-debian"]

    contexts = {
        debian_base = "docker-image://postgres:16-bookworm"
    }

    args = {
        CLANG_VERSION = "15"
    }
}

target "builder-ubuntu-jammy" {
    inherits = ["docker-metadata-action", "base-builder-debian"]

    contexts = {
        debian_base = "docker-image://ubuntu:jammy"
    }

    args = {
        CLANG_VERSION = "15"
    }
}

target "builder-ubuntu-noble" {
    inherits = ["docker-metadata-action", "base-builder-debian"]

    contexts = {
        debian_base = "docker-image://ubuntu:noble"
    }

    args = {
        CLANG_VERSION = "17"
    }
}

target "runner-postgres-16" {
    inherits = ["docker-metadata-action", "base-runner"]

    contexts = {
        postgres_base = "docker-image://postgres:16-bookworm"
    }
}

target "runner-postgres-15" {
    inherits = ["docker-metadata-action", "base-runner"]

    contexts = {
        postgres_base = "docker-image://postgres:15-bookworm"
    }
}

target "runner-postgres-14" {
    inherits = ["docker-metadata-action", "base-runner"]

    contexts = {
        postgres_base = "docker-image://postgres:14-bookworm"
    }
}

target "runner-postgres-13" {
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

target "base-builder-debian" {
    contexts = {
        pgxman = "target:pgxman"
    }

    dockerfile = "dockerfiles/builder/Dockerfile.debian"
}

target "base-runner" {
    contexts = {
        pgxman = "target:pgxman"
    }

    dockerfile = "dockerfiles/runner/Dockerfile.postgres"
}

target "pgxman" {
    dockerfile = "dockerfiles/shared/Dockerfile.pgxman"
    target = "gobuild"

    args = {
        BUILD_VERSION = "${BUILD_VERSION}"
    }
}

# Inherit this target for CI use
# Ref https://github.com/docker/metadata-action#bake-definition
target "docker-metadata-action" {}
