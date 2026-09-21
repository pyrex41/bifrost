\\ hash-eq.shen -- equal Shen values must hash equal (port-performance.md).
\\ Prints a single token so CLI eval-print quoting cannot split the golden.
(define hash= X Y Bound -> (= (hash X Bound) (hash Y Bound)))

(do (print (and (hash= true (intern "true") 1009)
                (and (hash= false (intern "false") 1009)
                     (and (hash= 1 1.0 1009)
                          (hash= 0 0.0 256)))))
    (nl))
