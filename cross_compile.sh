#!/bin/bash

dir=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)

mkdir -p $dir/out/
rm -f $dir/out/*

platforms=(
    "linux/arm"
    "linux/amd64"
    "linux/arm64"
    "linux/386"
    "darwin/amd64"
    "darwin/arm64"
    "windows/386"
    "windows/amd64"
    "windows/arm64")

projects=(
    "cli"
    "backend")

VER="$(go run ./utils/print_version.go)"
RELEASE_NOTES_FILE="$dir/out/RELEASE_NOTES.txt"
RELEASE_NOTES_LINK="https://github.com/benbusby/yeetfile/releases/download/v$VER"

cat >$RELEASE_NOTES_FILE <<EOL
___

Linux, macOS, and Windows binaries for the CLI app and for the server are included below. These archives contain a single executable that you can run on your machine.

EOL

capitalize()
{
  printf '%s' "$1" | head -c 1 | tr [:lower:] [:upper:]
  printf '%s' "$1" | tail -c '+2'
}

for project in "${projects[@]}"
do
    project_name=$project
    if [ $project = "cli" ]; then
        project_name=$(echo $project_name | tr '[a-z]' '[A-Z]')
    else
        project_name=$(capitalize $project)
    fi

    printf -- "### $project_name\n\n" >> $RELEASE_NOTES_FILE
    for platform in "${platforms[@]}"
    do
        echo "Compiling $project for $platform..."
        platform_split=(${platform//\// })
        GOOS=${platform_split[0]}
        GOARCH=${platform_split[1]}

        os_name=$(capitalize $GOOS)
        arch_name=$GOARCH

        if [ $project = "cli" ]; then
            output_name="yeetfile"
        else
            output_name="yeetfile-server"
        fi

        if [ $GOARCH = "arm" ]; then
            arch_name="arm32"
        fi

        compressed_name="${output_name}_${GOOS}_${arch_name}_${VER}.tar.gz"
        if [ $GOOS = "darwin" ]; then
            os_name="macOS"
            compressed_name="${output_name}_macos_${arch_name}_${VER}.tar.gz"
        elif [ $GOOS = "windows" ]; then
            compressed_name="${output_name}_windows_${arch_name}_${VER}.zip"
        fi

        if [ $GOOS = "windows" ]; then
            output_name+=".exe"
        fi

        compile_cmd="GOOS=$GOOS GOARCH=$GOARCH go build -ldflags='-s -w' -o $output_name ./$project"
        echo "└ $compile_cmd"
        eval $compile_cmd
        if [ $? -ne 0 ]; then
            echo "An error has occurred! Aborting the script execution..."
            exit 1
        fi

        if [ $GOOS = "windows" ]; then
            zip -j out/$compressed_name $output_name
        else
            tar -czvf out/$compressed_name $output_name
        fi
        rm -f $output_name

        full_link="$RELEASE_NOTES_LINK/$compressed_name"

        printf -- "- $os_name (\`$arch_name\`): [$tar_name]($full_link)\n" >> $RELEASE_NOTES_FILE
    done

    printf "\n" >> $RELEASE_NOTES_FILE
done

cat $RELEASE_NOTES_FILE
