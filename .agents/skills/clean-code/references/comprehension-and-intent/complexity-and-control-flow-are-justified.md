# Complexity and control flow are justified

The solution has no more abstraction, branching, indirection, or special-case handling than the problem actually requires.

Good design resists both needless cleverness and uncontrolled branching. A solution is maintainable when its complexity is proportional to the real problem, its abstractions are earned by repeated need, and its control flow is not dominated by one-off cases or cascades of conditions. These concerns belong together because each is a different way complexity grows faster than the requirement has earned.

Strong signs include a small number of moving parts, abstractions that remove real repetition or protect stable concepts, a main path that is easy to identify, and conditional logic that is structurally contained rather than scattered. Weak signs include speculative wrappers or interfaces, ceremony added for imagined future use, proliferating edge-case branches, deeply nested logic, interacting flags, and designs where understanding behavior means reconstructing a hidden exception model.

This symptom matters because excessive complexity compounds. It raises review cost, hides defects, makes tests harder to trust, and turns later changes into archaeology. Review should ask whether the structure is carrying real domain load or whether the code has accumulated indirection and branching that mostly serves itself.

## Examples

### Bad

```text
function shippingFee(order):
  if order.total >= 50:
    if order.isMember:
      return 0
    else:
      return 10
  else:
    return 10
```

### Good

```text
function shippingFee(order):
  if order.total >= 50 and order.isMember:
    return 0
  return 10
```
