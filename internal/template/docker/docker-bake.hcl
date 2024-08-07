variable "REPO" {
    default = "ghcr.io/pgxman/builder"
}

variable "TAG" {
    default = "main"
}

target "export" {
    contexts = {
        {{- if .ExportDebianBookwormArtifacts }}
        debian-bookworm = "target:debian-bookworm"
        {{- end }}
        {{- if .ExportUbuntuJammyArtifacts }}
        ubuntu-jammy = "target:ubuntu-jammy"
        {{- end }}
        {{- if .ExportUbuntuNobleArtifacts }}
        ubuntu-noble = "target:ubuntu-noble"
        {{- end }}
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

target "ubuntu-noble" {
    args = {
        BUILD_IMAGE = "${REPO}/ubuntu/noble:${TAG}"
    }

    dockerfile = "Dockerfile"
    target = "build"
}
