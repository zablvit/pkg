# common

This is a fork of https://github.com/gitops-tools/image-updater, a shared repository for tooling for interacting with Git.

Until merged, this is being maintained as a separated project. The main reason for the fork was the need to support additional operations (key and file removal) and drivers (mainly bitbucket), plus some minor changes for additional parsing of options from the env etc.

In more detail, besides the added functionality for file and key deletion (and the general usage as a yaml updater rather than just an image updater), the main change was swapping jenkins-x/go-scm with a fork of drone/go-scm. The [fork](https://github.com/ocraviotto/go-scm) merged the client factory interface from jenkins-x into drone to make it easier to swap it in here, and was necessary to provide better support for bitbucket and bitbucket cloud (the original project was mainly supporting gitlab and guthub). 

This is used as a library with the also forked and modified [yaml-updater](https://github.com/ocraviotto/yaml-updater) (originally named image-updater)

## Original notice

This is alpha code, just extracted from another project for reuse.
