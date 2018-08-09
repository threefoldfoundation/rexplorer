#!/bin/bash
set -e

# version is supplied as argument
version="$(git describe | cut -d '-' -f 1)"
commit="$(git rev-parse --short HEAD)"

if [ "$commit" == "$(git rev-list -n 1 $version | cut -c1-8)" ]
then
	full_version="$version"
else
	full_version="${version}-${commit}"
fi

for os in darwin linux; do
	echo Packaging ${os}...
	# create workspace
	folder="release/rexplorer-${version}-${os}-amd64"
	rm -rf "$folder"
	mkdir -p "$folder"
	# compile binary
	GOOS=${os} go build -a \
			-ldflags="-X main.rawVersion=${full_version} -s -w" \
			-o "${folder}/rexplorer" .
	# add other artifacts
	cp -r release_notes LICENSE README.md "$folder"
	# zip
	(
		zip -rq "release/rexplorer-${version}-${os}-amd64.zip" \
			"release/rexplorer-${version}-${os}-amd64"
	)
done