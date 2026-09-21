\\ properties.shen -- put/get on *property-vector* (S42 absvector store).
(let V (value *property-vector*)
  (do (put bifrost-prop-key bifrost-prop-attr 42 V)
      (do (print (get bifrost-prop-key bifrost-prop-attr V))
          (nl))))
