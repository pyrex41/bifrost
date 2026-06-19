\\ A minimal stand-in for a real project's self-test entrypoint.
\\
\\ The shape Bifrost expects from a third-party suite:
\\   * load your sources (here there are none -- a real project would
\\     `(load "load.shen")` first),
\\   * run your checks and print DETERMINISTIC output (identical across ports),
\\   * end by printing a success MARKER ("ALL PASS") iff everything passed.
\\
\\ Bifrost then drives this file on every supported port and asserts the
\\ normalised stdout is byte-identical AND contains the marker.
(define run-tests
  -> (let R1 (= (+ 2 3) 5)
          R2 (= (reverse [1 2 3]) [3 2 1])
          R3 (= (append [1 2] [3 4]) [1 2 3 4])
          R4 (= (map (lambda X (* X X)) [1 2 3]) [1 4 9])
          (if (and R1 (and R2 (and R3 R4)))
              (do (output "tiny-suite: 4/4 passed~%")
                  (output "ALL PASS~%")
                  true)
              (do (output "tiny-suite: SOME FAIL~%")
                  false))))

(run-tests)
