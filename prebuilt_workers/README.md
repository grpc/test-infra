# Using prebuilt images with gRPC OSS benchmarks

The scripts in this folder:
* Build the images have pre-built workers embedded.
* Push the images to specified registry.
* Delete the images from specified registry.

## Build and push images

The script [prepare_prebuilt_workers.go](pre_built_workers/prepare_prebuilt_workers.go) 
build images and push them to a user specified registry. 
The following example build and push prebuilt cxx and go worker images.
```
go run ./prebuilt_workers/prepare_prebuilt_workers/prepare_prebuilt_workers.go \
 -l cxx:master -l go:master \
 -p gcr.io/grpc-testing/e2etesting/pre_built_workers \
 -t user-specified-tag
```

The built `cxx` and `go` images contains the workers built from commit/branch wish 
to test. 
The script `prepare_for_prebuilt_workers.go` takes the following options:
* `-l `<br> Language and GITREF to benchmark. The language and its specific 
GITREF wish to build workers from can be specified as `language:gitref`.
May be repeated.
* `-t` <br> Tag for prebuilt images. The tag uniquely identify the images built
during current batch of tests. If no tag provided, the script provides a 
default tag in the format of:`test-initiator-YYYY-MM-DD-HH-MM-SS`, if the test 
initiator is not found, the `test-initiator` in the tag is replaced by 
`anonymous-user`. Tag complies with 
[docker tag's restrictions](https://docs.docker.com/engine/reference/commandline/tag/#extended-description). 
* `-p` <br> Image registry tp store images. In the example, the location of the
images are:
  ```
  gcr.io/grpc-testing/e2etesting/pre_built_workers/cxx:user-specified-tag
  gcr.io/grpc-testing/e2etesting/pre_built_workers/go:user-specified-tag
  ```

## Delete the images
The script [delete_prebuilt_workers.go](prebuilt_workers/delete_prebuilt_workers.go) 
deletes images within user specified registry. The script lists all images
within the specified registry, then check if the image has the user specified 
tag.
The following example delete all images within 
`gcr.io/grpc-testing/e2etesting/pre_built_workers` that have 
tag:`user-specified-tag`.
```
go run ./prebuilt_workers/delete_prebuilt_workers/delete_prebuilt_workers.go \
 -p gcr.io/grpc-testing/e2etesting/pre_built_workers \
 -t user-specified-tag
```
* `-t` <br> Tag for prebuilt images. Tag is the unique identifier for images to 
delete. If the image have multiple tags including the user specified
tag, the tag specified here is removed from image's tag list, otherwise the 
image is deleted.
* `-p` <br> Image registry to search images from. Only take image registry
without the image name. The image registry should also be the most direct 
registry of images, the directories within image registry will not be checked.