#!/usr/bin/env bash

test_description="Test rotate command"

. lib/test-lib.sh

test_init_ipfs

test_expect_success "Save first ID and key" '
ipfs id -f="<id>" > first_id &&
ipfs id -f="<pubkey>" > first_key
'

test_launch_ipfs_daemon

test_kill_ipfs_daemon

test_expect_success "rotating keys" '
ipfs rotate
'

test_expect_success "Compare second ID and key to first" '
ipfs id -f="<id>" > second_id &&
ipfs id -f="<pubkey>" > second_key &&
! test_cmp first_id second_id &&
! test_cmp first_key second_key
'

test_expect_success "checking ID" '
  ipfs config Identity.PeerID > expected-id &&
  ipfs id -f "<id>\n" > actual-id &&
  ipfs key list -l | grep self | cut -d " " -f1 > keystore-id &&
  test_cmp expected-id actual-id &&
  test_cmp expected-id keystore-id
'

test_launch_ipfs_daemon

test_kill_ipfs_daemon

test_done
