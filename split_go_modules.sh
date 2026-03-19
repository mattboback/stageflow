#!/bin/bash
set -e

cd libs/go
MODULES=(bootstrap config domain events httputil logging messaging models scannercatalog scannerregistry storage)

for MOD in "${MODULES[@]}"; do
    cd "$MOD"
    go mod init "github.com/mattboback/stageflow/libs/go/$MOD"
    
    # Replace references to packages/contracts
    go mod edit -replace github.com/mattboback/stageflow/packages/contracts/scanner-manifest=../../contracts/scanner-manifest
    go mod edit -replace github.com/mattboback/stageflow/packages/contracts/report/generated/go=../../contracts/report/generated/go
    
    # Replace references to other modules in libs/go
    for OTHER_MOD in "${MODULES[@]}"; do
        if [ "$MOD" != "$OTHER_MOD" ]; then
            go mod edit -replace "github.com/mattboback/stageflow/libs/go/$OTHER_MOD=../$OTHER_MOD"
        fi
    done
    cd ..
done

# Update imports in all files
find . -name "*.go" -type f -exec sed -i 's|github.com/mattboback/stageflow/packages/shared-go|github.com/mattboback/stageflow/libs/go|g' {} +

# Run go mod tidy for each module
for MOD in "${MODULES[@]}"; do
    cd "$MOD"
    go mod tidy
    cd ..
done

rm go.mod go.sum
