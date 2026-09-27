\\ load-echo-probe.shen -- probe whether (load ...) echoes each toplevel form's
\\ evaluated value, as the canonical kernel load requires.
\\
\\ klambda/load.kl:  (defun shen.eval-and-print (Forms)
\\                      (map (lambda F (pr (shen.app (eval-kl (shen.shen->kl F))
\\                                          "\n" shen.s) (stoutput))) Forms))
\\ i.e. load PRINTS each toplevel result before returning `loaded`.
\\
\\ Canonical impls, including shen-lua on both cold and warm fasl-cache paths,
\\ print: "PROBE" then 42 then (fn p). shen-lua replays the per-form echoes
\\ on a warm cache hit (pyrex41/shen-lua#41).
"PROBE"
(+ 40 2)
(define p -> ok)
