\\ fib.shen -- naive recursion; (fib 20) = 6765
(define fib
  0 -> 0
  1 -> 1
  N -> (+ (fib (- N 1)) (fib (- N 2))))

(do (print (fib 20)) (nl))
