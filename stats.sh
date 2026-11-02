#!/usr/bin/env bash

echo "Total"
cat results/pinned.json | jq 'length'

echo "Fully pinned"
cat results/pinned.json | jq 'map(select(.actions_total != 0 and .actions_total == .actions_pinned)) | length'

echo "Partially pinned"
cat results/pinned.json | jq 'map(select(.actions_total != 0 and .actions_pinned != 0)) | length'

echo "Not pinned"
cat results/pinned.json | jq 'map(select(.actions_total != 0 and .actions_pinned == 0)) | length'

echo "Not using actions"
cat results/pinned.json | jq 'map(select(.actions_total == 0)) | length'
