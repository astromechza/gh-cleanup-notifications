# gh-cleanup-notifications

A small binary I maintain for myself to cleanup my github notifications using my own custom heuristics. I generally run
this once or twice a day to help stay on top of all the automated notifications coming through in a busy organisation.

Written in Go because that works nicely with the `gh` cli.

In general, this does:

1. list a page of notifications
2. for each of (1), perform some initial filtering
3. for each of (2), lookup the notification subject (PR, issue, release, CI, etc..) and extract it's state
4. filter by heuristic function
5. mark the notification thread as read

## Usage

```
gh login
gh extension install astromechza/gh-cleanup-notifications
gh cleanup-notifications
```

## Development

- run 'cd gh-cleanup-notifications; gh extension install .; gh cleanup-notifications' to see your new extension in action
- run 'go build && gh cleanup-notifications' to see changes in your code as you develop
- run 'gh repo create' to share your extension with others

For more information on writing extensions:
https://docs.github.com/github-cli/github-cli/creating-github-cli-extensions

## References

- https://github.com/awendt/gh-cleanup-notifications
- https://github.com/rnorth/gh-tidy-notifications
- etc.
