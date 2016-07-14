#!/bin/bash -e

ginkgo \
  -r \
  -race \
  -randomizeSuites \
  -randomizeAllSpecs \
  -slowSpecThreshold=30 \
  -keepGoing \
  -cover \
  -skipPackage release_tests \
  $@
