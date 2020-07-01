#!/usr/bin/env bash

test_description="Test rotate command"

. lib/test-lib.sh

test_init_ipfs

test_launch_ipfs_daemon

test_kill_ipfs_daemon

test_expect_success "rotating keys" '
ipfs rotate
'

test_launch_ipfs_daemon

test_kill_ipfs_daemon

test_done
