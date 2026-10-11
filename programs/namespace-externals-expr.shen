\\ namespace-externals-expr.shen -- 42.2 namespaces: (externals EXPR) evaluates
\\ EXPR to a symbol list (42.1 accepted only a literal list). bifrost-ns-ext
\\ stays global; local becomes bifrost.demo.local.
\\ Ports that do not boot extension-namespaces print the BIFROST-SKIP marker.
(define bifrost-ns-vocab -> [bifrost-ns-ext])

(define bifrost-ns-run ->
  (do (map (/. F (eval F))
           (read-from-string "(shen.x.namespace bifrost.demo (externals (bifrost-ns-vocab)) (define bifrost-ns-ext X -> (+ X 1)) (define local X -> (bifrost-ns-ext X)))"))
      [(bifrost-ns-ext 41) (bifrost.demo.local 1)]))

(if (trap-error (do (shen.x.namespaces.initialise) true) (/. E false))
    (do (print (bifrost-ns-run)) (nl))
    (output "BIFROST-SKIP: namespaces extension not booted~%"))
