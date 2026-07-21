\\ load-echo-probe.shen -- probe whether (load ...) echoes each toplevel form's
\\ evaluated value, as the canonical kernel load requires.
\\
\\ klambda/load.kl:  (defun shen.eval-and-print (Forms)
\\                      (map (lambda F (pr (shen.app (eval-kl (shen.shen->kl F))
\\                                          "\n" shen.s) (stoutput))) Forms))
\\ i.e. load PRINTS each toplevel result before returning `loaded`.
\\
\\ Canonical impls (shen-cl / shen-rust / shen-go, and shen-lua with a COLD fasl
\\ cache or SHEN_FASL=off) print:  "PROBE"  then  42  then  (fn p).
\\ shen-lua on a WARM fasl-cache hit DROPS all three (boot.lua ~796; the
\\ documented "replayed load does not echo per-form values" degrade). That makes
\\ (load file) stdout depend on cache state -- a cross-port divergence and an
\\ intra-port nondeterminism. See cases/divergences.json load-toplevel-echo; pyrex41/shen-lua#40.
"PROBE"
(+ 40 2)
(define p -> ok)
