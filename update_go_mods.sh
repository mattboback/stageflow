#!/bin/bash
set -e

ROOT_DIR=$(pwd)
MODULES=(bootstrap config domain events httputil logging messaging models scannercatalog scannerregistry storage)

find services clients devtools qa libs -name "go.mod" -type f | while read -r modfile; do
    dir=$(dirname "$modfile")
    cd "$dir"
    
    # Calculate relative path to root
    REL_PATH=$(realpath --relative-to="." "$ROOT_DIR")

    # Replace packages/contracts -> libs/contracts
    sed -i 's|packages/contracts|libs/contracts|g' go.mod
    
    # Replace packages/shared-go -> libs/go
    sed -i 's|packages/shared-go|libs/go|g' go.mod
    
    # If the go.mod had `replace github.com/mattboback/stageflow/libs/go => ...`
    # We should remove it because libs/go is no longer a single module.
    sed -i '/replace github.com\/mattboback\/stageflow\/libs\/go =>/d' go.mod
    
    # Add replaces for the split modules
    for MOD in "${MODULES[@]}"; do
        go mod edit -replace "github.com/mattboback/stageflow/libs/go/$MOD=$REL_PATH/libs/go/$MOD"
    done
    
    # Add replaces for contracts
    go mod edit -replace "github.com/mattboback/stageflow/libs/contracts/scanner-manifest=$REL_PATH/libs/contracts/scanner-manifest"
    go mod edit -replace "github.com/mattboback/stageflow/libs/contracts/report/generated/go=$REL_PATH/libs/contracts/report/generated/go"
    
    go mod tidy || true
    
    cd "$ROOT_DIR"
done
