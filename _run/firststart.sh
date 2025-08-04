#!/bin/bash

scripts/git.sh --add_commit
scripts/git.sh --add_push

cd ../

mkdir -p target
mkdir -p tmp


#go work sync
bash "_run/scripts/go_tidy_all.sh"
go generate
