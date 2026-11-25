# Two-sum

Implement `two_sum` in `glue/two_sum_functions.py`:

    def two_sum(nums: list[int], target: int) -> list[int]:
        """Return the indices of the two numbers that add up to target."""

The parameter names, order and type annotations are fixed: the wrapper server
in `glue/` checks them against the OCaml signature and refuses to start when
they have drifted.

Rules:

- exactly one solution exists, so there is no ambiguity to resolve
- an element may not be used twice, even when its value appears more than once
- the two indices may be returned in any order
- `nums` is not sorted and may hold negative values

Examples:

    two_sum([2, 7, 11, 15], 9)  ->  [0, 1]
    two_sum([3, 2, 4], 6)       ->  [1, 2]
    two_sum([3, 3], 6)          ->  [0, 1]

The checks the chapter runs against the implementation live next to the
signature in `glue/two_sum.ml`.
