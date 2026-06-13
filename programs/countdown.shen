\\ countdown.shen -- tail-recursive loop; must not overflow the stack.
\\ Prints "done" when it reaches 0.
(define countdown
  0 -> done
  N -> (countdown (- N 1)))

(do (print (countdown 100000)) (nl))
