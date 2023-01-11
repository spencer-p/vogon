# Vogon - todo.txt productivity with vim

This is a nifty Vim package and Go utility to easily manage a list of todo.txt
compatible lines with markdown headings.

[![asciicast](https://asciinema.org/a/jlMk1Ot5cE37qBSWuDT5VS9vx.svg)](https://asciinema.org/a/jlMk1Ot5cE37qBSWuDT5VS9vx)

## Why?

This project is the natural result of a few key things I've noticed:

1. Planning and organizing is best for me when I can make quick and fast edits.
1. I edit text with vim faster than anything else.
1. I prefer to let formatters do the dirty work - and a lot of task management
   is dirty work.

Vogon leverages vim and a smart formatter to create a practical and effective
project management system in my editor, which I can interact with like any
other block of text.

The formatter shares a lot in common with a compiler - it reads the current
state of the world, lexes, parses, crunches, and emits a result. It's also a lot
like the classic
[model-view-controller](https://en.wikipedia.org/wiki/Model%E2%80%93view%E2%80%93controller)
architecture, embraced by [Elm](https://guide.elm-lang.org/architecture/),
React, and other TUI libraries like
[bubbletea](https://github.com/charmbracelet/bubbletea). I think it's natural
and satisfying to extend the core concept of a code formatter to a UI
controller.

Vogon allows me to achieve a few things that I think are really great for
productivity:

1. Clear groupings, starting with the **Inbox**, where I can get things off my
   mind and into the machine quickly.
1. Scheduling. All I have to do is add `sched:<some date>`, or even just `s:t`,
   to automatically move a task to the schedule or today's list. Tasks
   automatically move into the today list on their scheduled date.
1. A home for every future action I've defined, in both **Next** and **Someday**
lists.
1. A **Logbook**, where I can refer to what I've accomplished as need be.
1. I can re-use my years of experience to work smarter.
1. Compatability with todo.txt (sort of) and its many tools.

It's not perfect. It has some rough edges, and it's only barely fast enough to
be snappy - it typically runs in just under 30ms. **But**: It makes me highly
effective, it makes me enjoy planning, and I can always hack on it. 🙂

## Installation

You'll need go and vim.

```bash
git clone https://github.com/spencer-p/vogon ~/.vim/pack/plugins/start/vogon
cd ~/.vim/pack/plugins/start/vogon
go install .
# Make sure your PATH includes your go install path.
```
