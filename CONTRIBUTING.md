![](docs/images/deluxe-circle.svg)

# Contributing to One-Nats

Want to hack on One-Nats? Awesome! We're excited to have you join us as we build great tools!

This page contains information about reporting issues as well as some tips and guidelines useful to experienced open source contributors. 

## Reporting security issues

One-Nats is meant to be run internally on machines and not out on the internet and hence may contain code not deemed to be safe.  

If you do find a security issue in One-Nats, please disclose it privately. Please don't file a GitHub issue - we will be sure to thank you for it publicly when the issue is fixed.

## Reporting other issues

A great way to get started with an open source project is to contribute issues. If you're running One-Nats and you encounter something that doesn't seem quite right, or maybe there's a feature that you think fits the tool well and doesn't exist yet, you should file an issue. Issues are found on the top tab of our GitHub page.

When filing an bug issue, please always include the One-Nats version and any steps that you took to encounter the problem. Please be a detailed as possible, since this will help everyone resolve the issue quickly.

## Quick contribution tips and guidelines


### Pull requests are always welcome

No issue is too small for a pull request. We love to get them and we appreciate your contribution. When you've got a change that you'd like to make, please create a GitHub issue for it. Send us your pull request and reference your issue in the comment. Any discussions will be tracked in the issue. Don't be discouraged if your pull request is rejected! We'll do our best to work with you to get your change added to the project.

### Conventions

Submit unit tests for your changes. Go has a great test framework built in; use
it! Take a look at existing tests for inspiration. Run the full test suite on your branch beforesubmitting a pull request.

Update the documentation when creating or modifying features. Test your
documentation changes for clarity, concision, and correctness, as well as a
clean documentation build.

Write clean code. Universally formatted code promotes ease of writing, reading,
and maintenance. 

### Commits

Commits are how we organize the history of our work, find things that happened in the past and even track down bugs. Please organize your commits into useful chunks of work with a short message describing what the contents of the commit will do when applied. For guidance on how to write commit messages see [How to Write a Git Commit Message](http://chris.beams.io/posts/git-commit/).


## Coding Style

The rules:

1. All code should be formatted with `gofmt -s`.
2. All code should pass the default levels of
   [`golint`](https://github.com/golang/lint).
3. All code should follow the guidelines covered in [Effective
   Go](http://golang.org/doc/effective_go.html) and [Go Code Review
   Comments](https://github.com/golang/go/wiki/CodeReviewComments).
4. Comment the code. Tell us the why, the history and the context.
5. Document _all_ declarations and methods, even private ones. Declare
   expectations, caveats and anything else that may be important. If a type
   gets exported, having the comments already there will ensure it's ready.
6. Variable name length should be proportional to its context and no longer.
   `noCommaALongVariableNameLikeThisIsNotMoreClearWhenASimpleCommentWouldDo`.
   In practice, short methods will have short variable names and globals will
   have longer names.
7. No underscores in package names. If you need a compound name, step back,
   and re-examine why you need a compound name. If you still think you need a
   compound name, lose the underscore.
8. No utils or helpers packages. If a function is not general enough to
   warrant its own package, it has not been written generally enough to be a
   part of a util package. Just leave it unexported and well-documented.
9. All tests should run with `go test` and outside tooling should not be
   required. No, we don't need another unit testing framework. Assertion
   packages are acceptable if they provide _real_ incremental value.
10. Even though we call these "rules" above, they are actually just
    guidelines. Since you've read all the rules, you now know that.

