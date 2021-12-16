#!/bin/bash

# inplace_envsubst injects environment variables to files passed as arguments
inplace_envsubst () {
  files=("$@")
  for file in "${files[@]}"; do
    tmp=$(mktemp)
    echo "Making substitutions in ${file}"
    cp --attributes-only --preserve "${file}" "${tmp}"
    envsubst < "${file}" > "$tmp" && mv "$tmp" "$file"
  done
}

inplace_envsubst "$@"
