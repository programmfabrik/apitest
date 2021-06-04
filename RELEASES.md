# v1.0.2

Added .github/workflows/release.yml .github/workflows/unit-tests.yml

* Added ci steps to release a binary
* Added checkout code steps before running the tests
* Improved automatic release
* Updated .github/workflows/release.yml
* Changed draft: true to draft: false

.circleci/config.yml:

* Removed circleci as ci executor

Updated Makefile:

* Improved GOOS and GOARCH handling
* Added ci target to Makefile

Updated README.md

* Changed status badges to match github-actions
