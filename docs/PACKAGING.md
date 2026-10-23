# Packaging

## Tagging a new upstream release

1. Edit the Version value in `elk.spec`
2. Tag the commit and push it

    ```shell
    $ VERSION=X.Y.Z
    $ git add elk.spec
    $ git commit -s -m "chore: v$VERSION"
    $ git tag -a -s -m v$VERSION v$VERSION
    $ git push origin v$VERSION
    ```

## Validating downstream packaging

```shell
$ make archive
$ make archive-deps
$ make srpm
$ make rpm [NOCHECK=1]
```
