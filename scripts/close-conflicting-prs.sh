#!/bin/bash
# Script to close conflicting Dependabot PRs
# Run this after authenticating with GitHub CLI: gh auth login

gh pr close 25 --reason "conflicting" --comment "Closing - conflicts with current main branch. Please resubmit after main is stable."
gh pr close 24 --reason "conflicting" --comment "Closing - conflicts with current main branch. Please resubmit after main is stable."
gh pr close 23 --reason "conflicting" --comment "Closing - conflicts with current main branch. Please resubmit after main is stable."
gh pr close 22 --reason "conflicting" --comment "Closing - conflicts with current main branch. Please resubmit after main is stable."
gh pr close 21 --reason "conflicting" --comment "Closing - conflicts with current main branch. Please resubmit after main is stable."
gh pr close 20 --reason "conflicting" --comment "Closing - conflicts with current main branch. Please resubmit after main is stable."
gh pr close 19 --reason "conflicting" --comment "Closing - conflicts with current main branch. Please resubmit after main is stable."
gh pr close 18 --reason "conflicting" --comment "Closing - conflicts with current main branch. Please resubmit after main is stable."
gh pr close 16 --reason "conflicting" --comment "Closing - conflicts with current main branch. Please resubmit after main is stable."
gh pr close 17 --reason "conflicting" --comment "Closing - conflicts with current main branch. Please resubmit after main is stable."

echo "Done closing PRs. A new push to main will trigger fresh Dependabot PRs."