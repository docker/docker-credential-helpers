variable "GO_VERSION" {
  default = "1.16.7"
}
variable "BIN_OUT" {
  default = "./bin"
}

target "_common" {
  args = {
    GO_VERSION = GO_VERSION
  }
}

group "default" {
  targets = ["binaries"]
}

group "validate" {
  targets = ["lint", "vendor-validate", "docs-validate"]
}

target "lint" {
  inherits = ["_common"]
  dockerfile = "./hack/dockerfiles/lint.Dockerfile"
  output = ["type=cacheonly"]
}

target "vendor-validate" {
  inherits = ["_common"]
  dockerfile = "./hack/dockerfiles/vendor.Dockerfile"
  target = "validate"
  output = ["type=cacheonly"]
}

target "vendor-update" {
  inherits = ["_common"]
  dockerfile = "./hack/dockerfiles/vendor.Dockerfile"
  target = "update"
  output = ["."]
}

target "test" {
  inherits = ["_common"]
  target = "test-coverage"
  output = ["./coverage"]
  platforms = [
//    "darwin/amd64",
//    "darwin/arm64",
    "linux/amd64",
//    "windows/amd64"
  ]
}

target "binaries" {
  inherits = ["_common"]
  target = "binaries"
  output = [BIN_OUT]
  platforms = [
    "darwin/amd64",
    "darwin/arm64",
    "linux/amd64",
    "windows/amd64"
  ]
}

target "deb" {
  inherits = ["binaries"]
  target = "deb"
  platforms = [
    "linux/amd64"
  ]
}
