\\ prolog.shen -- a small Shen Prolog query.
\\ Define a parent/2 relation and ask for a grandparent.
(defprolog parent
  abe homer <-- ;
  homer bart <-- ;
  homer lisa <--;)

(defprolog grandparent
  X Z <-- (parent X Y) (parent Y Z);)

\\ Collect grandchildren of abe. Expect: [bart lisa]
(do (print (prolog? (grandparent abe Who) (return Who))) (nl))
