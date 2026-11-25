(* The signature below is the source of truth for the two_sum function the
   candidate implements in two_sum_functions.py. After changing it, regenerate
   the wrapper server and commit both glue files:

     dune exec utf -- gen python glue/two_sum.ml

   `dune runtest` below builds the bridge and runs the checks. *)

module Two_sum : sig
  val two_sum : nums:int list -> target:int -> int list
end = struct end [@@utest]

let sorted = List.sort compare

let string_of_ints xs = "[" ^ String.concat "; " (List.map string_of_int xs) ^ "]"

let failures = ref 0

let check name expected actual =
  if expected = actual then Printf.printf "ok   %s\n" name
  else (
    incr failures;
    Printf.printf "FAIL %s: expected %s, got %s\n" name (string_of_ints expected)
      (string_of_ints actual))

let () =
  check "adds up to the target" [ 0; 1 ]
    (sorted (Two_sum.two_sum ~nums:[ 2; 7; 11; 15 ] ~target:9));
  check "the answer is not the first pair" [ 1; 2 ]
    (sorted (Two_sum.two_sum ~nums:[ 3; 2; 4 ] ~target:6));
  check "equal values do not reuse an index" [ 0; 1 ]
    (sorted (Two_sum.two_sum ~nums:[ 3; 3 ] ~target:6));
  check "negative values count" [ 0; 3 ]
    (sorted (Two_sum.two_sum ~nums:[ -3; 4; 3; 90 ] ~target:87));
  if !failures > 0 then (
    Printf.printf "%d check(s) failed\n" !failures;
    exit 1);
  print_endline "all two-sum checks passed"
