Who ever wants a documentation if there is a gif?

![gif explain basics](https://cloud.githubusercontent.com/assets/674812/4304381/4135dc04-3e70-11e4-8bcd-979fc8bd4946.gif)

Installation
============

Now `ash` is available only from `go get`:

```
go get github.com/seletskiy/ash
```

After that, `ash` executable should be available for use.

Important note
==============

`ash` is in a development phase, so it can crash unexpectedly.

*However*, all data you've entered will *not* be lost and will kept even
in case of crash.

If you experience a crash, *please*, fill an issue and attach `ash` debug
output (using `--debug=2` cmd line flag) and review file caused a crash.

Why
===

There is a couple points:

* Stash web-interface is, well, slow; it will eventually slow down when your
  review grows in number of comments; your editor probably will not;
* Sometimes Stash can lost comments you're entered in spite of having "Drafts";
* You can unleash all power of your editor in reviewing code by highlighting
  syntax, inserting snippets and even running code;

Usage
=====

Authorization
-------------

First of all, you need to specify login parameters for accessing Stash.

Easiest way to do that is to create config file named `~/.config/ash/ashrc`
and add following lines:

```
--user
  <your username here>

--pass
  <your password here>
```

Setting your editor
-------------------

To make `ash` use editor of your choice you need either to export `$EDITOR`
environment variable or just pass editor name to the `-e` flag.

You can use `ashrc` config too:

```
[... some other options goes here ...]

-e
 vim
```

Running ash
-----------

Now, using `ash` you can:

* list files in the review;
* review concrete file;
* see recent changes in overview mode;

Common usage of `ash` is:

```
ash inbox (only if --host given)
ash <pull request url> ls
ash <pull request url> review
ash <pull request url> review <file to review>
```

Reviewing
---------

You can modify review file which will be opened in editor by:

* entering new line comments just after line you want to comment; you need to
  use `#` prefix for these kind of lines;
* modifying existing comments by just altering their text in file;
* deleting existing comments by just deleting their body;
* adding review-level/file-level comments by entering them outside of the diff
  context;
* replying to the existing comments by entering reply lines of text with some
  indentation *after* comment delimiter `---`;

Tips and tricks
---------------

You can use shorthand syntax for accessing reviews without specifying full PR
url.

There are two flags for that:
* `--host` which used to specify Stash host (e.g. http://stash.local/);
* `--project` which used to specify default project to search repo/pull-request;

So, you can add following to your `ashrc`:
```
--host
  http://<your stash hostname>/

--project
  mycoolproject
```

Now you can run `ash` like this:
```
ash myrepo/123 review
ash notsocoolproject/anotherrepo/456 review
```

State of things
===============

* [x] fill todo list;
* [x] list files in review;
* [x] add, modify, delete line comments as well as review/file level comments;
* [x] make `ash` work in overview mode;
* [x] list reviews in project;
* [x] list inbox;
* [ ] integrate `ash` with `vim` using `Unite` (PR is welcomed);
* [ ] integrate `ash` with `sublime` writing a plugin (PR is welcomed);
* [ ] be more tolerant to user mistakes (`ash` can crash sometime);
* [ ] wrap long lines in comments;
