# pullem

This small utility walks a directory tree looking for repos (directories with `.git` subdirs in them) and updates them.
Before an update can happen it checks that

* repo is on its default branch (usually that's master)
* repo is clean (no changes or untracked files)
* repo can be fast-forwarded
* repo has a remote called `origin`

If these conditions hold it does a `git pull origin master --ff-only` on it.
 
