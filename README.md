# lazyreview

Review GitHub pull requests on your own machine, in a terminal.

One command loads a pull request, a stack of pull requests, or a branch with
no pull request yet. lazyreview then shows a class diagram of the change,
every diff, the threads already on GitHub and the comments you queue. One
press posts your review. Ask Claude about a line, and the answer stays beside
it.

![A stack of two pull requests in the terminal app, with a conversation with Claude in the diff](docs/images/terminal-review.png)

## What you need

- Node.js 22 or later
- git, and the GitHub command line `gh`, signed in
- Go 1.24 or later, for the terminal app
- Claude Code (`claude`), signed in, only to ask about a line

lazyreview has no packages to install. Its few libraries are in the
repository, so it reaches no network except through `gh`, git and Claude
Code. It is tested on macOS.

## Start

    git clone https://github.com/connorb645/lazyreview
    cd lazyreview
    bin/lazyreview load https://github.com/owner/repo/pull/12

`load` prints the review's address, such as
`lazyreview://github.com/owner/repo/12`. lazyreview runs no server: the
terminal app reads and writes the data home itself.

To load a stack, name each pull request, bottom first. To load a branch with
no pull request, name the folder, the base and the branch:

    bin/lazyreview load --repo ~/code/widgets --base main --head feature/search

To open a review in the terminal app, give it the address, or the same
link:

    bin/lazyreview terminal lazyreview://github.com/owner/repo/12

The terminal app is laid out as in Lazygit, with Vim keys. `Space ?` lists
every key.

With no link, the terminal app opens on a home screen. The home screen lists
the open pull requests that involve your GitHub accounts, by repository. Name
the accounts in the config, `config.yml`, as
[the guide](docs/guide.md#the-home-screen) shows; `lazyreview config path`
says where it is.

![The home screen: open pull requests by repository](docs/images/terminal-home.png)

## More

[The guide](docs/guide.md) covers stacks, written reviews, every key of the
terminal app, components and packages, and where the data is kept.
[Config](docs/Config.md) lists every setting of the config.

Run the tests with `npm test`, and the terminal app's tests with
`npm run test:terminal`. The tests reach no network.

In a clone, `make run` opens the terminal app on the home screen, and
`make run URL=<address>` opens the review at an address `load` or `list`
printed.

To land a finished branch, run `make land BRANCH=<branch>` in a clone on
`main`. It checks first, then moves `main` to the branch on every remote and
in the clone, fast-forward only. `lazyreview stop` stops every answer Claude
is still giving.

## Licence

MIT. The libraries in `vendor/` and `terminal/vendor/` keep their own
licences.
