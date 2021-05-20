# Using prebuilt images with gRPC OSS benchmarks

The scripts in this folder:
* Build the images with the worker executables compiled and embedded.
* Push the images to specified registry.
* Delete the images from specified registry.

## Build and push images

The script [prepare_prebuilt_workers.go](pre_built_workers/prepare_prebuilt_workers.go) 
builds images and pushes them to a user specified Google clooud registry. For example, the following shows this process (building and pushing prebuilt images) for cxx and go workers:
```
go run test-infra/tools/prepare_prebuilt_workers/prepare_prebuilt_workers.go \
 -l cxx:master \
 -l go:master \
 -p gcr.io/grpc-testing/project-name/pre_built_workers \
 -t user-specified-tag \
 -r test-infra/containers/pre_built_workers
```

These built `cxx` and `go` images contain the workers built from commit/branch we wish 
to test. 

The script `prepare_for_prebuilt_workers.go` takes the following options:
* `-l `<br> Language and GITREF to benchmark. The language and its specific 
GITREF to build the workers from can be specified as `language:COMMIT_SHA`.
May be repeated.
* `-t` <br> Tag for prebuilt images. Tag is a required fields. Tag complies with 
[docker tag's restrictions](https://docs.docker.com/engine/reference/commandline/tag/#extended-description). 
* `-r` <br> Root directory of Dockerfiles.
* `-build-only` <br> Option to build only, when specified 
  `-build-only=true`, images will only be built locally.
* `-p` <br> Image registry to store images. The helper script only support pushing images to GCR. If another image registry is chosen, the images could still be built through script with `-build-only=true`, the user could then push the images manually. The Dockerfiles that the script building from is avalible in [test-infra/containers/pre_built_images](test-infra/containers/pre_built_images). If using Google cloud registry, the image prefix should be in form of `gcr.io/project-name/your-directory-name`. In the example, the built images would be stored as: 
 :
  ```
  gcr.io/grpc-testing/project-name/pre_built_workers/cxx:user-specified-tag
  gcr.io/grpc-testing/project-name/pre_built_workers/go:user-specified-tag
  ```

## Delete the images

The script [delete_prebuilt_workers.go](prebuilt_workers/delete_prebuilt_workers.go) 
deletes images within user specified registry. The script lists all images
within the specified registry, then check if the image has the user specified 
tag. This script only supports Google Container Registry, because it relies on the google-cloud-sdk.

The following example delete all images within 
`gcr.io/grpc-testing/project-name/pre_built_workers` that have 
tag:`user-specified-tag`.

```
go run test-infra/tools/delete_prebuilt_workers/delete_prebuilt_workers.go \
 -p gcr.io/grpc-testing/project-name/pre_built_workers \
 -t user-specified-tag
```

* `-t` <br> Tag for prebuilt images. Tag is the unique identifier for images to 
delete. If the image have multiple tags including the user specified
tag, the tag specified here is removed from image's tag list, otherwise the 
image is deleted.
* `-p` <br> Image registry to search images from. Only take image registry
without the image name. The image registry should also be the most direct 
registry of images, the directories within image registry will not be checked.
