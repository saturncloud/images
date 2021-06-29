#!/bin/bash

# [description]
#     Checks the saturn* image directories to be sure they
#     use standard practices. Saturn has some application logic
#     that relies on the images in this repo as a source of truth
#     for metadata about images. This script runs on every build to
#     ensure that the directories are in a state that Saturn will
#     understand.

set -eou pipefail

echo "----- checking that image metadata follows Saturn practices ----"

images_to_check=$(cat .ci/images-to-lint.txt)

error_count=0

for image in ${images_to_check}; do
    echo ""
    echo "${image}"

    # directory should exist
    if [ -d "${image}" ]; then
        echo "  * directory '${image}' exists"
    else
        echo "  * [ISSUE] directory '${image}' not found"
        error_count=$((error_count + 1))
        continue
    fi

    # environment.yml should exist
    conda_env_file="${image}/environment.yml"
    if [ -f "${conda_env_file}" ]; then
        echo "  * environment.yml exists"
    else
        echo "  * [ISSUE] environment.yml not found"
        error_count=$((error_count + 1))
    fi

    # environment.yml should not use a 'prefix:' block
    if [ $(grep --count -E "^prefix" ${conda_env_file}) -gt 0 ]; then
        echo "  * [ISSUE] found 'prefix:' in ${conda_env_file}"
        error_count=$((error_count + 1))
    else
        echo "  * environment.yml does not have 'prefix:'"
    fi

    # Dockerfile should not include extra 'pip install' or 'conda install' stuff,
    # as these wouldn't be read correctly by Saturn
    dockerfile="${image}/Dockerfile"

    for package_manager in pip conda; do
        if [ $(grep --count -E "${package_manager} install" ${dockerfile}) -gt 0 ]; then
            echo "  * [ISSUE] found '${package_manager} install' in ${dockerfile}. Update ${conda_env_file} instead."
            error_count=$((error_count + 1))
        else
            echo "  * no ${package_manager} installs found in Dockerfile"
        fi

        for installer_file in ${image}/*.bash; do
            if [ -f "${installer_file}" ]; then
                if [ $(grep -v mamba ${installer_file} | grep --count -E "${package_manager} install") -gt 0 ]; then
                    echo "  * [ISSUE] found '${package_manager} install' in ${installer_file}. Update ${conda_env_file} instead."
                    error_count=$((error_count + 1))
                else
                    echo "  * no ${package_manager} installs found in ${installer_file}"
                fi
            fi
        done
    done

done

echo ""
if [ ${error_count} -gt 0 ]; then
    echo "----- issues found while checking images: ${error_count} -------"
else
    echo "----- No issues found in standard images. ----------------------"
fi

exit ${error_count}
