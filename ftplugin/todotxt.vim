if exists('b:did_ftplugin')
  finish
endif
let b:did_ftplugin = 1

setlocal foldmethod=syntax
setlocal foldlevel=20
setlocal textwidth=0

set autoread
autocmd BufWritePre todo.txt call TodoTxtFmt()

function! TodoTxtFmt() abort
let l:curw = winsaveview()
%!vogon -f -
call winrestview(l:curw)
endfunction
