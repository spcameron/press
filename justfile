# print this help message
[default]
help:
    @just --list

# initialize a cloned starter repo
[group('*workflow')]
init app module:
    @scripts/init {{app}} {{module}}

# runs audit and unit tests
[group('*workflow')]
check:
    @scripts/audit
    @scripts/test unit

# runs audit and race tests
[group('*workflow')]
ci:
    @scripts/audit
    @scripts/test race

# runs tidy and go-mod-upgrade
[group('*workflow')]
maintain:
    @scripts/tidy
    @scripts/mod-upgrade

# runs a chain of QC commands
[group('quality')]
audit:
    @scripts/audit

# runs tidy and formatting
[group('quality')]
tidy:
    @scripts/tidy

# runs go-mod-upgrade
[group('quality')]
mod-upgrade:
    @scripts/mod-upgrade

# runs tests, accepts commands [unit|race|cover]
[group('test')]
test mode="":
    @scripts/test {{mode}}

# sync main and delete local branch (for branches with no PR)
[group('git')]
branch-delete:
    @scripts/git/branch-delete

# sync main and create a new branch
[group('git')]
branch-create:
    @scripts/git/branch-create

# create PR for current branch
[group('git')]
pr-create:
    @scripts/git/pr-create

# merge PR for local branch
[group('git')]
pr-merge:
    @scripts/git/pr-merge

# merge PR for local branch, sync main, and delete local branch
[group('git')]
pr-finish:
    @scripts/git/pr-merge --cleanup

# view open PR in browser
[group('git')]
pr-view:
    @scripts/git/pr-view

# push local branch and set upstream to origin
[group('git')]
push-upstream:
    @scripts/git/push-upstream

# rebase branch onto origin/main
[group('git')]
rebase-main:
    @scripts/git/rebase-main

# rebase branch onto its configured upstream
[group('git')]
rebase-upstream:
    @scripts/git/rebase-upstream

# backup current main and reset main to origin/main
[group('git')]
repair-main:
    @scripts/git/repair-main

# rebase branch onto upstream then origin/main, audit, and publish
[group('git')]
sync-branch:
    @scripts/git/sync-branch

# alias for sync-branch
[group('git')]
sync:
    @scripts/git/sync-branch

# fast-forward main from origin/main
[group('git')]
sync-main:
    @scripts/git/sync-main
