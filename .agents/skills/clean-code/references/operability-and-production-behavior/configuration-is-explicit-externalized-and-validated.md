# Configuration is explicit, externalized, and validated

Environment-dependent behavior is configured outside core logic, read in controlled places, and checked early enough that the system does not run on broken assumptions.

Configuration is maintainable when it is both separate from domain logic and disciplined in how it enters the system. Values that vary by environment, deployment, tenant, or operating mode should not be hard-coded into the implementation. At the same time, configuration should not be read casually throughout the codebase. It should enter through controlled edges, be interpreted in a small number of places, and be validated before the system starts doing meaningful work.

Strong signs include configuration defined outside source, a clear composition or startup boundary where config is loaded, explicit mapping from raw settings to typed internal concepts, and early failure when required values are missing, malformed, or contradictory. Weak signs include environment lookups scattered through business code, magic defaults that hide misconfiguration, runtime surprises caused by late config parsing, and systems that continue half-broken because critical assumptions were never checked.

This symptom matters because configuration errors often look like logic errors in production. Externalizing, centralizing, and validating config reduces environmental drift, makes deployments safer, and keeps the trusted core of the system focused on business behavior rather than process state. It also gives reviewers a clear place to inspect how operating assumptions enter the code.

## Examples

### Bad

```text
function sendReceipt(order):
  endpoint = environment["RECEIPT_API"] ?? "http://receipt.internal"
  timeout = integer(environment["RECEIPT_TIMEOUT"])
  post(endpoint, order, timeout)
```

### Good

```text
function start(environment):
  config = ReceiptConfig(requiredUrl(environment, "RECEIPT_API"),
                         requiredDuration(environment, "RECEIPT_TIMEOUT"))
  require(config.timeout > 0)
  run(ReceiptService(config))
```
