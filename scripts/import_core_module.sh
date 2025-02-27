#!/usr/bin/env bash
# SPDX-License-Identifier: LGPL-3.0-only

CORE_REPO="https://github.com/EdgeMatrixChain/edge-matrix-core.git"
CORE_TAG="v1.0.3"
CORE_DIR="./edge-matrix-core"

set -eux

case $TARGET in
	"build")
		git clone $CORE_REPO $CORE_DIR
    pushd $CORE_DIR
    git checkout $CORE_TAG

    popd
		;;

esac
