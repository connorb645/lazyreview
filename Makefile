# make run                    open the terminal app on the home screen
# make run URL=<address>      open the review at that address, as load and
#                             list print it, instead
# make land BRANCH=<branch>   land a finished branch on main: push it to main
#                             on every remote, fast-forward only, and bring
#                             this checkout's main to it
# make restart                what an older make land runs once it has
#                             landed: stops every answer Claude is still
#                             giving, and a server an older lazyreview left
#                             running
#
# Each works on the repository this file is in, from any folder.

root := $(dir $(abspath $(lastword $(MAKEFILE_LIST))))

.PHONY: run land restart

run:
	@"$(root)bin/lazyreview" terminal $(if $(URL),"$(URL)")

land:
	@go -C "$(root)terminal" run -mod=vendor ./cmd/land "$(BRANCH)"

restart:
	@"$(root)bin/lazyreview" stop
