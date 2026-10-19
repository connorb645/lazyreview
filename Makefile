# make land BRANCH=<branch>   land a finished branch on main: push it to main
#                             on every remote, fast-forward only, bring this
#                             checkout's main to it, then restart the app
# make restart                stop the app and start it again
#
# Both work on the repository this file is in, from any folder.

root := $(dir $(abspath $(lastword $(MAKEFILE_LIST))))

.PHONY: land restart

land:
	@node "$(root)scripts/land.mjs" "$(BRANCH)"
	@$(MAKE) --no-print-directory -f "$(root)Makefile" restart

restart:
	@"$(root)bin/lazyreview" stop
	@"$(root)bin/lazyreview" start
