#!/usr/bin/env fish

# Test configuration and initialization

set -e TWINE_BASE_DIRS
set output (source conf.d/twine.fish 2>&1)
@test "warns when TWINE_BASE_DIRS not configured" (string match -q "*TWINE_BASE_DIRS not configured*" -- $output) $status -eq 0

set -e TWINE_SESSION_PREFIX
source conf.d/twine.fish 2>/dev/null
@test "sets default TWINE_SESSION_PREFIX to empty" (test "$TWINE_SESSION_PREFIX" = "") $status -eq 0

set -e TWINE_USE_TMUXINATOR
source conf.d/twine.fish 2>/dev/null
@test "sets default TWINE_USE_TMUXINATOR to auto" (test "$TWINE_USE_TMUXINATOR" = "auto") $status -eq 0

set -gx TWINE_USE_TMUXINATOR false
source conf.d/twine.fish 2>/dev/null
@test "__twine_use_tmuxinator returns 1 when explicitly disabled" (__twine_use_tmuxinator) $status -eq 1

set -gx TWINE_USE_TMUXINATOR true
source conf.d/twine.fish 2>/dev/null
set -l old_PATH $PATH
set -gx PATH /nonexistent
@test "__twine_use_tmuxinator returns 1 when tmuxinator not installed" (__twine_use_tmuxinator) $status -eq 1
set -gx PATH $old_PATH

set -gx TWINE_USE_TMUXINATOR true
source conf.d/twine.fish 2>/dev/null
set -l temp_bin (mktemp -d)
touch $temp_bin/tmuxinator
chmod +x $temp_bin/tmuxinator
set -l old_PATH $PATH
set -gx PATH $temp_bin $PATH
@test "__twine_use_tmuxinator returns 0 when explicitly enabled and tmuxinator exists" (__twine_use_tmuxinator) $status -eq 0
set -gx PATH $old_PATH
rm -rf $temp_bin

set -gx TWINE_USE_TMUXINATOR auto
set -gx TWINE_TMUXINATOR_LAYOUT dev
source conf.d/twine.fish 2>/dev/null
set -l temp_bin (mktemp -d)
touch $temp_bin/tmuxinator
chmod +x $temp_bin/tmuxinator
set -l old_PATH $PATH
set -gx PATH $temp_bin $PATH
@test "__twine_use_tmuxinator auto mode returns 0 when tmuxinator exists and layout configured" (__twine_use_tmuxinator) $status -eq 0
set -gx PATH $old_PATH
rm -rf $temp_bin

set -gx TWINE_USE_TMUXINATOR auto
set -e TWINE_TMUXINATOR_LAYOUT
source conf.d/twine.fish 2>/dev/null
set -l temp_bin (mktemp -d)
touch $temp_bin/tmuxinator
chmod +x $temp_bin/tmuxinator
set -l old_PATH $PATH
set -gx PATH $temp_bin $PATH
@test "__twine_use_tmuxinator auto mode returns 1 when layout not configured" (__twine_use_tmuxinator) $status -eq 1
set -gx PATH $old_PATH
rm -rf $temp_bin
