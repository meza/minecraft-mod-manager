# End-to-end tests protect critical journeys

End-to-end tests cover a small number of high-value user or system journeys through the deployed shape of the application.

End-to-end tests are expensive evidence and should be spent on flows that matter.
They should prove that critical journeys work across the system, not exhaustively retest every validation branch or internal rule through the slowest path.
Strong signs include stable user-visible selectors, independent test data, controlled external dependencies, minimal unrelated setup through the UI, and useful artifacts such as traces, screenshots, logs, or correlation identifiers when failures occur.
Weak signs include arbitrary sleeps, CSS-selector fragility, global shared accounts, ordering dependence between tests, uncontrolled third-party calls, and large UI suites that duplicate lower-level coverage.
This symptom matters because brittle end-to-end suites train teams to rerun instead of investigate.
Review should ask whether each end-to-end test protects a critical journey that is worth its cost.

## Examples

### Bad
```text
E2E "customer completes checkout"
  customer = sharedCustomer("checkout-user")
  checkout uses livePaymentProvider
  click ".product:nth-child(7) .add"
  enter shared test card details
  click "#checkout .submit"
  sleep 3000
  assert confirmation and every checkout rule
```

### Good
```text
E2E "customer completes checkout"
  customer = createIndependentCustomer()
  product = createAvailableProduct()
  checkout uses controlledApprovedPaymentProvider
  customer signs in using role "Sign in"
  customer adds product using role "Add to cart"
  customer enters sandbox card details
  customer uses role "Place order"
  assert order confirmation shows the new order
```
