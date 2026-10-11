\\ source-form-order.shen -- 42.1 source-form handlers run BEFORE ordinary
\\ macros, and macros still see what the handler produced.
\\ bifrost-probe has both a source-form handler (-> bifrost-probe2) and a
\\ macro rule (-> macro-first). Handler-first prints handler-then-macro.
\\ Ports that do not boot the hook print the BIFROST-SKIP capability marker.
(define bifrost-sf-handler
  [bifrost-probe X] -> [[bifrost-probe2 X]])

(defmacro bifrost-sf-macro
  [bifrost-probe X] -> [@p macro-first X]
  [bifrost-probe2 X] -> [@p handler-then-macro X])

(define bifrost-sf-run ->
  (do (shen.register-source-form bifrost-probe (fn bifrost-sf-handler))
      (map (/. F (eval F)) (read-from-string "(bifrost-probe 1) (bifrost-probe 2)"))))

(if (trap-error (do (fn shen.register-source-form) true) (/. E false))
    (do (print (bifrost-sf-run)) (nl))
    (output "BIFROST-SKIP: source-form handlers not booted~%"))
