" Vim syntax file " Language: todo.txt
"
if !exists('g:main_syntax')
  if v:version < 600
    syntax clear
  elseif exists('b:current_syntax')
    finish
  endif
  let g:main_syntax = 'todo.txt'
endif

highlight TodoToday	term=bold cterm=bold ctermfg=Black ctermbg=DarkYellow
highlight TodoEve	term=bold cterm=bold ctermfg=Black ctermbg=DarkMagenta
highlight TodoInbox	term=bold cterm=bold ctermfg=Black ctermbg=DarkCyan
highlight TodoNext	term=bold cterm=bold ctermfg=Black ctermbg=DarkGreen
highlight TodoSched	term=bold cterm=bold ctermfg=Black ctermbg=DarkRed
highlight TodoLog	term=bold cterm=bold ctermfg=Black ctermbg=DarkBlue
highlight TodoHeader	term=bold cterm=bold ctermfg=White ctermbg=Black
highlight TodoContext	ctermfg=Green

syntax match header	'^# .*$'	contains=today,inbox,next,sched,log,eve
syntax keyword today	contained Today
syntax keyword inbox	contained Inbox
syntax keyword next		contained Next
syntax keyword sched	contained Scheduled
syntax keyword log		contained Logged
syntax keyword eve		contained Evening

syntax match project	'\(^\|\W\)+[^[:blank:]]\+'	contains=NONE
syntax match context	'\(^\|\W\)@[^'[:blank:]]\+'	contains=NONE
syntax match date		'\d\{2,4\}-\d\{2\}-\d\{2\}' contains=NONE
syntax match complete	'^x\>'						contains=NONE
syntax match specialTag	'\(^\|\W\)[^[:blank:]]\+:[^[:blank:]]\+'	contains=NONE

highlight default link today	TodoToday
highlight default link eve	TodoEve
highlight default link inbox	TodoInbox
highlight default link next		TodoNext
highlight default link sched	TodoSched
highlight default link log		TodoLog
highlight default link header	TodoHeader

highlight default link project	Keyword
highlight default link context	Label
highlight default link date		Comment
highlight default link complete	Delimiter
highlight default link specialTag		Comment

syntax region todoFold start='^#' end=/^#/me=s-2 transparent fold
