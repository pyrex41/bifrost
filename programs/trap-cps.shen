\\ trap-cps.shen -- trap-error that uses E, and freeze captured by a lambda.
(let Trap (trap-error (simple-error "boom") (/. E (error-to-string E)))
  (let Frozen ((let G (freeze 7) (/. Y (thaw G))) 0)
    (do (print (and (= Trap "boom") (= Frozen 7)))
        (nl))))
