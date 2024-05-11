#!/bin/bash

go build -o pinned-actions .
scp pinned-actions rita:/home/datosh/pinned-actions
rm pinned-actions
ssh rita
