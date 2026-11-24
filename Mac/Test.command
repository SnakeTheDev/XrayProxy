#!/bin/bash
# Re-run the Home-Reality self-test - double-click this file.
if [ -f "$HOME/.homereality/selftest.sh" ]; then bash "$HOME/.homereality/selftest.sh"; else echo "Not installed yet - run Install.command first."; fi
echo; read -n 1 -s -r -p "Press any key to close..."; echo
