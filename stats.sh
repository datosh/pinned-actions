#!/bin/bash

echo "Total"
cat result.json | jq 'length'

echo "Fully pinned"
cat result.json | jq 'map(select(.actions_total != 0 and .actions_total == .actions_pinned)) | length'

echo "Partially pinned"
cat result.json | jq 'map(select(.actions_total != 0 and .actions_pinned != 0)) | length'

echo "Not pinned"
cat result.json | jq 'map(select(.actions_total != 0 and .actions_pinned == 0)) | length'

echo "Not using actions"
cat result.json | jq 'map(select(.actions_total == 0)) | length'
