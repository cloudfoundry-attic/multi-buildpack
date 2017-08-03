#!/usr/bin/env bash
set -euo pipefail
set -x

export ROOT=$(dirname $(readlink -f ${BASH_SOURCE%/*}))
if [ ! -f "$ROOT/.bin/ginkgo" ]; then
  (cd "$ROOT/src/compile/vendor/github.com/onsi/ginkgo/ginkgo/" && go install)
fi

GINKGO_NODES=${GINKGO_NODES:-2}
GINKGO_ATTEMPTS=${GINKGO_ATTEMPTS:-2}

cd $ROOT/src/compile/integration
ginkgo --flakeAttempts=$GINKGO_ATTEMPTS -nodes $GINKGO_NODES
